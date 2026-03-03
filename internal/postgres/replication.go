package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PublicationOptions contains options for creating a publication.
type PublicationOptions struct {
	// Name is the publication name
	Name string
	// Tables is a list of tables to include (empty means all tables)
	Tables []string
	// AllTables publishes all tables in the database
	AllTables bool
	// Schemas limits ALL TABLES to specific schemas (PostgreSQL 15+)
	Schemas []string
	// PublishOperations specifies which operations to publish (insert, update, delete, truncate)
	PublishOperations []string
}

// SubscriptionOptions contains options for creating a subscription.
type SubscriptionOptions struct {
	// Name is the subscription name
	Name string
	// SourceConnString is the connection string to the source database
	SourceConnString string
	// PublicationName is the name of the publication to subscribe to
	PublicationName string
	// CopyData determines whether to copy existing data
	CopyData bool
	// CreateSlot determines whether to create a replication slot
	CreateSlot bool
	// SlotName is the replication slot name (defaults to subscription name)
	SlotName string
	// Enabled determines whether the subscription starts enabled
	Enabled bool
}

// SubscriptionStatus represents the status of a subscription.
type SubscriptionStatus struct {
	Name            string
	Enabled         bool
	SlotName        string
	PublicationName string
	ReceivedLSN     string
	LatestEndLSN    string
	LastMsgSendTime *time.Time
	LastMsgRecvTime *time.Time
	LatestEndTime   *time.Time
	ReplicationLag  *time.Duration
}

// TableReplicationState represents the replication state of a single table.
type TableReplicationState struct {
	SchemaName string
	TableName  string
	State      string // 'i' = initializing, 'r' = ready, 'd' = syncing data
	LSN        string
}

// PreflightCheck represents results of pre-flight checks for replication.
type PreflightCheck struct {
	WALLevel                 string
	WALLevelOK               bool
	MaxReplicationSlots      int
	SlotsAvailable           int
	HasReplicationPermission bool
	Extensions               []string
}

// OpenConnection opens a PostgreSQL connection.
func OpenConnection(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// RunPreflightChecks performs pre-flight checks.
func RunPreflightChecks(ctx context.Context, db *sql.DB) (*PreflightCheck, error) {
	check := &PreflightCheck{}

	var walLevel string
	err := db.QueryRowContext(ctx, "SHOW wal_level").Scan(&walLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to check wal_level: %w", err)
	}
	check.WALLevel = walLevel
	check.WALLevelOK = walLevel == "logical"

	// Check max_replication_slots
	var maxSlots int
	err = db.QueryRowContext(ctx, "SHOW max_replication_slots").Scan(&maxSlots)
	if err != nil {
		return nil, fmt.Errorf("failed to check max_replication_slots: %w", err)
	}
	check.MaxReplicationSlots = maxSlots

	// Count active replication slots
	var activeSlots int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pg_replication_slots").Scan(&activeSlots)
	if err != nil {
		return nil, fmt.Errorf("failed to count replication slots: %w", err)
	}
	check.SlotsAvailable = maxSlots - activeSlots

	// Check replication permission
	var hasReplication bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_roles
			WHERE rolname = current_user
			AND (rolreplication = true OR rolsuper = true)
		)
	`).Scan(&hasReplication)
	if err != nil {
		// If query fails, assume we might have permission
		check.HasReplicationPermission = true
	} else {
		check.HasReplicationPermission = hasReplication
	}

	// Get installed extensions (exclude plpgsql as it's built-in)
	rows, err := db.QueryContext(ctx, `
		SELECT extname
		FROM pg_extension
		WHERE extname != 'plpgsql'
		ORDER BY extname
	`)
	if err != nil {
		// If query fails, just skip extension check
		check.Extensions = []string{}
	} else {
		defer rows.Close()
		var extensions []string
		for rows.Next() {
			var extname string
			if err := rows.Scan(&extname); err == nil {
				extensions = append(extensions, extname)
			}
		}
		check.Extensions = extensions
	}

	return check, nil
}

// CreatePublication creates a publication.
func CreatePublication(ctx context.Context, db *sql.DB, opts PublicationOptions) error {
	// If schemas are specified but no specific tables, query for tables in those schemas
	if opts.AllTables && len(opts.Schemas) > 0 && len(opts.Tables) == 0 {
		var tables []string
		for _, schema := range opts.Schemas {
			schemaQuery := `
				SELECT schemaname, tablename
				FROM pg_tables
				WHERE schemaname = $1
				ORDER BY tablename
			`
			rows, err := db.QueryContext(ctx, schemaQuery, schema)
			if err != nil {
				return fmt.Errorf("failed to query tables in schema %s: %w", schema, err)
			}
			defer rows.Close()

			for rows.Next() {
				var schemaName, tableName string
				if err := rows.Scan(&schemaName, &tableName); err != nil {
					return fmt.Errorf("failed to scan table name: %w", err)
				}
				// Build schema-qualified table name without quotes for the list
				// We'll handle quoting when building the query
				tables = append(tables, schemaName+"."+tableName)
			}
			if err := rows.Err(); err != nil {
				return fmt.Errorf("error iterating tables: %w", err)
			}
		}
		opts.Tables = tables
		opts.AllTables = false
	}

	var query strings.Builder
	query.WriteString(fmt.Sprintf("CREATE PUBLICATION %s", QuoteIdentifier(opts.Name)))

	if opts.AllTables {
		query.WriteString(" FOR ALL TABLES")
	} else if len(opts.Tables) > 0 {
		query.WriteString(" FOR TABLE ")
		for i, table := range opts.Tables {
			if i > 0 {
				query.WriteString(", ")
			}
			// Handle schema-qualified table names (schema.table)
			if strings.Contains(table, ".") {
				parts := strings.SplitN(table, ".", 2)
				query.WriteString(QuoteIdentifier(parts[0]) + "." + QuoteIdentifier(parts[1]))
			} else {
				query.WriteString(QuoteIdentifier(table))
			}
		}
	} else {
		query.WriteString(" FOR ALL TABLES")
	}

	if len(opts.PublishOperations) > 0 {
		query.WriteString(fmt.Sprintf(" WITH (publish = '%s')", strings.Join(opts.PublishOperations, ", ")))
	}

	_, err := db.ExecContext(ctx, query.String())
	if err != nil {
		return fmt.Errorf("failed to create publication: %w", err)
	}

	return nil
}

// DropPublication drops a publication.
func DropPublication(ctx context.Context, db *sql.DB, name string, ifExists bool) error {
	query := "DROP PUBLICATION "
	if ifExists {
		query += "IF EXISTS "
	}
	query += QuoteIdentifier(name)

	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop publication: %w", err)
	}

	return nil
}

// CreateSubscription creates a subscription.
func CreateSubscription(ctx context.Context, db *sql.DB, opts SubscriptionOptions) error {
	var query strings.Builder
	query.WriteString(fmt.Sprintf("CREATE SUBSCRIPTION %s CONNECTION '%s' PUBLICATION %s",
		QuoteIdentifier(opts.Name),
		escapeSingleQuotes(opts.SourceConnString),
		QuoteIdentifier(opts.PublicationName),
	))

	// Build WITH options
	var withOpts []string
	withOpts = append(withOpts, fmt.Sprintf("copy_data = %t", opts.CopyData))
	withOpts = append(withOpts, fmt.Sprintf("create_slot = %t", opts.CreateSlot))
	withOpts = append(withOpts, fmt.Sprintf("enabled = %t", opts.Enabled))
	if opts.SlotName != "" {
		withOpts = append(withOpts, fmt.Sprintf("slot_name = '%s'", opts.SlotName))
	}

	query.WriteString(fmt.Sprintf(" WITH (%s)", strings.Join(withOpts, ", ")))

	_, err := db.ExecContext(ctx, query.String())
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	return nil
}

// DisableSubscription disables a subscription.
func DisableSubscription(ctx context.Context, db *sql.DB, name string) error {
	query := fmt.Sprintf("ALTER SUBSCRIPTION %s DISABLE", QuoteIdentifier(name))
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to disable subscription: %w", err)
	}
	return nil
}

// EnableSubscription enables a subscription.
func EnableSubscription(ctx context.Context, db *sql.DB, name string) error {
	query := fmt.Sprintf("ALTER SUBSCRIPTION %s ENABLE", QuoteIdentifier(name))
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to enable subscription: %w", err)
	}
	return nil
}

// DropSubscription drops a subscription from the database.
func DropSubscription(ctx context.Context, db *sql.DB, name string, ifExists bool) error {
	query := "DROP SUBSCRIPTION "
	if ifExists {
		query += "IF EXISTS "
	}
	query += QuoteIdentifier(name)

	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop subscription: %w", err)
	}

	return nil
}

// GetSubscriptionStatus retrieves the status of a subscription.
func GetSubscriptionStatus(ctx context.Context, db *sql.DB, name string) (*SubscriptionStatus, error) {
	status := &SubscriptionStatus{}

	// Get subscription info
	err := db.QueryRowContext(ctx, `
		SELECT
			s.subname,
			s.subenabled,
			s.subslotname,
			s.subpublications[1]
		FROM pg_subscription s
		WHERE s.subname = $1
	`, name).Scan(&status.Name, &status.Enabled, &status.SlotName, &status.PublicationName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("subscription %q not found", name)
		}
		return nil, fmt.Errorf("failed to get subscription status: %w", err)
	}

	// Get replication status from pg_stat_subscription
	var receivedLSN, latestEndLSN sql.NullString
	var lastMsgSendTime, lastMsgRecvTime, latestEndTime sql.NullTime
	err = db.QueryRowContext(ctx, `
		SELECT
			received_lsn::text,
			latest_end_lsn::text,
			last_msg_send_time,
			last_msg_receipt_time,
			latest_end_time
		FROM pg_stat_subscription
		WHERE subname = $1
		LIMIT 1
	`, name).Scan(&receivedLSN, &latestEndLSN, &lastMsgSendTime, &lastMsgRecvTime, &latestEndTime)
	if err != nil && err != sql.ErrNoRows {
		// Non-critical error, continue without replication stats
	}

	if receivedLSN.Valid {
		status.ReceivedLSN = receivedLSN.String
	}
	if latestEndLSN.Valid {
		status.LatestEndLSN = latestEndLSN.String
	}
	if lastMsgSendTime.Valid {
		status.LastMsgSendTime = &lastMsgSendTime.Time
	}
	if lastMsgRecvTime.Valid {
		status.LastMsgRecvTime = &lastMsgRecvTime.Time
	}
	if latestEndTime.Valid {
		status.LatestEndTime = &latestEndTime.Time
	}

	// Calculate replication lag
	if status.LastMsgSendTime != nil && status.LastMsgRecvTime != nil {
		lag := status.LastMsgRecvTime.Sub(*status.LastMsgSendTime)
		status.ReplicationLag = &lag
	}

	return status, nil
}

// GetTableReplicationStates retrieves the replication state of all tables for a subscription.
func GetTableReplicationStates(ctx context.Context, db *sql.DB, subName string) ([]TableReplicationState, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			srrelid::regclass::text,
			srsubstate,
			COALESCE(srsublsn::text, '')
		FROM pg_subscription_rel sr
		JOIN pg_subscription s ON sr.srsubid = s.oid
		WHERE s.subname = $1
		ORDER BY srrelid::regclass::text
	`, subName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table replication states: %w", err)
	}
	defer rows.Close()

	var states []TableReplicationState
	for rows.Next() {
		var state TableReplicationState
		var fullName string
		err := rows.Scan(&fullName, &state.State, &state.LSN)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table state: %w", err)
		}

		// Parse schema.table format
		parts := strings.SplitN(fullName, ".", 2)
		if len(parts) == 2 {
			state.SchemaName = parts[0]
			state.TableName = parts[1]
		} else {
			state.SchemaName = "public"
			state.TableName = fullName
		}

		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table states: %w", err)
	}

	return states, nil
}

// ListSubscriptions lists all subscriptions in the database.
func ListSubscriptions(ctx context.Context, db *sql.DB) ([]SubscriptionStatus, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			s.subname,
			s.subenabled,
			s.subslotname,
			s.subpublications[1]
		FROM pg_subscription s
		ORDER BY s.subname
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []SubscriptionStatus
	for rows.Next() {
		var status SubscriptionStatus
		err := rows.Scan(&status.Name, &status.Enabled, &status.SlotName, &status.PublicationName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription: %w", err)
		}
		subs = append(subs, status)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating subscriptions: %w", err)
	}

	return subs, nil
}

// CreateReplicationSlot creates a logical replication slot using SQL.
func CreateReplicationSlot(ctx context.Context, db *sql.DB, slotName string) error {
	_, err := db.ExecContext(ctx, "SELECT pg_create_logical_replication_slot($1, 'pgoutput')", slotName)
	if err != nil {
		return fmt.Errorf("failed to create replication slot: %w", err)
	}
	return nil
}

// DropReplicationSlot drops a replication slot on the source database.
func DropReplicationSlot(ctx context.Context, db *sql.DB, slotName string) error {
	_, err := db.ExecContext(ctx, "SELECT pg_drop_replication_slot($1)", slotName)
	if err != nil {
		return fmt.Errorf("failed to drop replication slot: %w", err)
	}
	return nil
}

// GetReplicationSlotLag retrieves the lag of a replication slot in bytes.
func GetReplicationSlotLag(ctx context.Context, db *sql.DB, slotName string) (int64, error) {
	var lag int64
	err := db.QueryRowContext(ctx, `
		SELECT pg_current_wal_lsn() - confirmed_flush_lsn
		FROM pg_replication_slots
		WHERE slot_name = $1
	`, slotName).Scan(&lag)
	if err != nil {
		return 0, fmt.Errorf("failed to get replication slot lag: %w", err)
	}
	return lag, nil
}

// QuoteIdentifier quotes a PostgreSQL identifier.
func QuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// escapeSingleQuotes escapes single quotes in a string.
func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// TableStateDescription returns a human-readable description of a table replication state.
func TableStateDescription(state string) string {
	switch state {
	case "i":
		return "initializing"
	case "d":
		return "copying data"
	case "f":
		return "finished copy"
	case "s":
		return "synchronized"
	case "r":
		return "ready"
	default:
		return "unknown"
	}
}
