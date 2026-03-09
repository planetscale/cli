package branch

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/postgres"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/roleutil"
)

// ImportStatusOutput represents the JSON output for import status.
type ImportStatusOutput struct {
	Subscription string                     `json:"subscription"`
	Enabled      bool                       `json:"enabled"`
	Publication  string                     `json:"publication"`
	ReceivedLSN  string                     `json:"received_lsn,omitempty"`
	Tables       []ImportTableStatusOutput  `json:"tables,omitempty"`
	Summary      *ImportStatusSummaryOutput `json:"summary,omitempty"`
}

// ImportTableStatusOutput represents the status of a single table.
type ImportTableStatusOutput struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	State  string `json:"state"`
	LSN    string `json:"lsn,omitempty"`
}

// ImportStatusSummaryOutput represents the summary of import status.
type ImportStatusSummaryOutput struct {
	Total        int `json:"total"`
	Ready        int `json:"ready"`
	Copying      int `json:"copying"`
	Initializing int `json:"initializing"`
}

func ImportStatusCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		publication  string
		subscription string
		dbname       string
		watch        bool
		interval     time.Duration
		format       printer.Format
	}

	cmd := &cobra.Command{
		Use:   "status <database> <branch>",
		Short: "Show the status of a PostgreSQL import",
		Long: `Show the status of an active PostgreSQL import, including subscription status
and table-by-table replication progress.

Use --watch to continuously monitor the import progress.`,
		Example: `  # Check import status
  pscale branch import status mydb main

  # Watch import progress
  pscale branch import status mydb main --watch

  # Watch with custom interval
  pscale branch import status mydb main --watch --interval 5s

  # Output as JSON
  pscale branch import status mydb main --format json`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if db.Kind != "postgresql" {
				return fmt.Errorf("database %s is not a PostgreSQL database", printer.BoldBlue(database))
			}

			creds, err := postgres.NewImportCredentials()
			if err != nil {
				return fmt.Errorf("failed to access credentials: %w", err)
			}

			subName, dbName, err := resolveSubscriptionName(ctx, creds, ch.Config.Organization, database, branch, flags.subscription, flags.dbname)
			if err != nil {
				return err
			}

			dstDB, cleanup, err := connectForStatus(ctx, client, creds, ch.Config.Organization, database, branch, subName, dbName)
			if cleanup != nil {
				defer cleanup()
			}
			if err != nil {
				return err
			}
			defer dstDB.Close()

			showStatus := func() error {
				status, tables, err := fetchImportStatus(ctx, dstDB, subName)
				if err != nil {
					return err
				}

				summary := calculateSummary(tables)

				if flags.format == printer.JSON {
					return formatStatusJSON(ch, status, tables, summary)
				}
				return printStatus(ch, database, branch, status, tables, summary, flags.watch, flags.interval)
			}

			if !flags.watch {
				return showStatus()
			}

			ticker := time.NewTicker(flags.interval)
			defer ticker.Stop()

			if err := showStatus(); err != nil {
				return err
			}

			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
					if err := showStatus(); err != nil {
						ch.Printer.Printf("Error: %v\n", err)
					}
				}
			}
		},
	}

	cmd.Flags().StringVar(&flags.publication, "publication", "", "Publication name (if not using stored value)")
	cmd.Flags().StringVar(&flags.subscription, "subscription", "", "Subscription name (for teammates without keychain access)")
	cmd.Flags().StringVar(&flags.dbname, "dbname", "postgres", "PostgreSQL database name on destination")
	cmd.Flags().BoolVar(&flags.watch, "watch", false, "Continuously watch import progress")
	cmd.Flags().DurationVar(&flags.interval, "interval", 2*time.Second, "Refresh interval for watch mode")
	cmd.Flags().Var(printer.NewFormatValue(printer.Human, &flags.format), "format", "Output format (human, json)")

	return cmd
}

func resolveSubscriptionName(ctx context.Context, creds *postgres.ImportCredentials, org, db, branch, flagSub, flagDB string) (sub, dbname string, err error) {
	if flagSub != "" {
		dbname = flagDB
		if dbname == "" {
			dbname = "postgres"
		}
		return flagSub, dbname, nil
	}

	subs, err := creds.ListStoredSubscriptions(org, db, branch)
	if err != nil {
		return "", "", fmt.Errorf("failed to list imports: %w", err)
	}

	if len(subs) == 0 {
		return "", "", fmt.Errorf("no active imports found for %s/%s\nHint: Use --subscription flag to specify the subscription name", db, branch)
	}

	if len(subs) > 1 {
		var selected string
		prompt := &survey.Select{
			Message: "Multiple imports found. Which one would you like to check?",
			Options: subs,
		}
		if err := survey.AskOne(prompt, &selected); err != nil {
			return "", "", err
		}
		sub = selected
	} else {
		sub = subs[0]
	}

	info, err := creds.GetImportInfoForSubscription(org, db, branch, sub)
	if err != nil {
		return "", "", fmt.Errorf("failed to retrieve import info: %w", err)
	}

	return sub, info.DBName, nil
}

func connectForStatus(ctx context.Context, client *ps.Client, creds *postgres.ImportCredentials, org, database, branch, subName, dbName string) (*sql.DB, func(), error) {
	info, err := creds.GetImportInfoForSubscription(org, database, branch, subName)
	if err == nil && info.RoleUsername != "" && info.RolePassword != "" && info.RoleHost != "" {
		dbName := info.DBName
		if dbName == "" {
			dbName = "postgres"
		}
		cfg := &postgres.Config{
			Host:     info.RoleHost,
			Port:     5432,
			User:     info.RoleUsername,
			Password: info.RolePassword,
			Database: dbName,
			SSLMode:  "require",
			Options:  make(map[string]string),
		}
		db, err := postgres.OpenConnection(postgres.BuildConnectionString(cfg))
		if err == nil {
			if err := db.PingContext(ctx); err == nil {
				return db, nil, nil
			}
			db.Close()
		}
	}

	role, err := createTempRole(ctx, client, org, database, branch)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temporary role: %w", err)
	}
	cfg := &postgres.Config{
		Host:     role.Role.AccessHostURL,
		Port:     5432,
		User:     role.Role.Username,
		Password: role.Role.Password,
		Database: dbName,
		SSLMode:  "require",
		Options:  make(map[string]string),
	}
	db, err := postgres.OpenConnection(postgres.BuildConnectionString(cfg))
	if err != nil {
		role.Cleanup(ctx, "postgres")
		return nil, nil, fmt.Errorf("failed to connect to destination: %w", err)
	}
	return db, func() { role.Cleanup(ctx, "postgres") }, nil
}

func fetchImportStatus(ctx context.Context, db *sql.DB, subName string) (*postgres.SubscriptionStatus, []postgres.TableReplicationState, error) {
	status, err := postgres.GetSubscriptionStatus(ctx, db, subName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get subscription status: %w", err)
	}

	tables, err := postgres.GetTableReplicationStates(ctx, db, subName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get table states: %w", err)
	}

	return status, tables, nil
}

func calculateSummary(states []postgres.TableReplicationState) *ImportStatusSummaryOutput {
	summary := &ImportStatusSummaryOutput{
		Total: len(states),
	}
	for _, t := range states {
		switch t.State {
		case "r":
			summary.Ready++
		case "d", "f", "s":
			summary.Copying++
		case "i":
			summary.Initializing++
		}
	}
	return summary
}

func formatStatusJSON(ch *cmdutil.Helper, status *postgres.SubscriptionStatus, tables []postgres.TableReplicationState, summary *ImportStatusSummaryOutput) error {
	output := ImportStatusOutput{
		Subscription: status.Name,
		Enabled:      status.Enabled,
		Publication:  status.PublicationName,
		ReceivedLSN:  status.ReceivedLSN,
		Summary:      summary,
	}
	for _, t := range tables {
		output.Tables = append(output.Tables, ImportTableStatusOutput{
			Schema: t.SchemaName,
			Table:  t.TableName,
			State:  postgres.TableStateDescription(t.State),
			LSN:    t.LSN,
		})
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	ch.Printer.Println(string(data))
	return nil
}

func printStatus(ch *cmdutil.Helper, database, branch string, status *postgres.SubscriptionStatus, tables []postgres.TableReplicationState, summary *ImportStatusSummaryOutput, watch bool, interval time.Duration) error {
	if watch {
		fmt.Print("\033[H\033[2J")
	}

	ch.Printer.Printf("%s\n\n", printer.Bold("Import Status"))
	ch.Printer.Printf("Subscription: %s\n", printer.BoldBlue(status.Name))

	enabledStr := printer.BoldGreen("enabled")
	if !status.Enabled {
		enabledStr = printer.BoldRed("disabled")
	}
	ch.Printer.Printf("Status: %s\n", enabledStr)
	ch.Printer.Printf("Publication: %s\n", status.PublicationName)

	if status.ReceivedLSN != "" {
		ch.Printer.Printf("Received LSN: %s\n", status.ReceivedLSN)
	}

	if status.ReplicationLag != nil {
		ch.Printer.Printf("Replication lag: %s\n", status.ReplicationLag.String())
	}

	ch.Printer.Printf("\n%s\n", printer.Bold("Table Progress:"))
	ch.Printer.Printf("  Ready: %d/%d", summary.Ready, summary.Total)
	if summary.Copying > 0 {
		ch.Printer.Printf(" | Copying: %d", summary.Copying)
	}
	if summary.Initializing > 0 {
		ch.Printer.Printf(" | Initializing: %d", summary.Initializing)
	}
	ch.Printer.Printf("\n")

	if summary.Total > 0 {
		pct := float64(summary.Ready) / float64(summary.Total) * 100
		barWidth := 40
		filled := int(pct / 100 * float64(barWidth))
		bar := strings.Repeat("=", filled) + strings.Repeat("-", barWidth-filled)
		ch.Printer.Printf("  [%s] %.1f%%\n", bar, pct)
	}

	var notReady []postgres.TableReplicationState
	for _, t := range tables {
		if t.State != "r" {
			notReady = append(notReady, t)
		}
	}

	if len(notReady) > 0 && len(notReady) <= 10 {
		ch.Printer.Printf("\n%s\n", printer.Bold("Tables in progress:"))
		for _, t := range notReady {
			ch.Printer.Printf("  %s.%s: %s\n", t.SchemaName, t.TableName, postgres.TableStateDescription(t.State))
		}
	} else if len(notReady) > 10 {
		ch.Printer.Printf("\n  ... and %d more tables in progress\n", len(notReady)-10)
	}

	if summary.Ready == summary.Total && summary.Total > 0 {
		ch.Printer.Printf("\n%s All tables are ready for replication!\n", printer.BoldGreen("✓"))
		ch.Printer.Printf("Run 'pscale branch import complete %s %s' to finish the import.\n", database, branch)
	}

	if watch {
		ch.Printer.Printf("\nLast updated: %s (refreshing every %s)\n",
			time.Now().Format("15:04:05"), interval)
	}

	return nil
}

func createTempRole(ctx context.Context, client *ps.Client, org, database, branch string) (*roleutil.Role, error) {
	return roleutil.New(ctx, client, roleutil.Options{
		Organization:   org,
		Database:       database,
		Branch:         branch,
		Name:           fmt.Sprintf("pscale_import_temp_%d", time.Now().Unix()),
		TTL:            10 * time.Minute,
		InheritedRoles: []string{"postgres"},
	})
}
