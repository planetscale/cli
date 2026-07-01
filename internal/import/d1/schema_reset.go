package d1

import (
	"context"
	"fmt"
	"strings"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/planetscale/cli/internal/postgres"
)

const (
	postgresRoleName = "postgres"
	publicSchemaName = "public"
)

func cleanupStaleImportRoles(ctx context.Context, psClient *ps.Client, opts ImportOptions, currentUsername string) error {
	if psClient == nil || opts.DestURI != "" {
		return nil
	}

	roles, err := psClient.PostgresRoles.List(ctx, &ps.ListPostgresRolesRequest{
		Organization: opts.Org,
		Database:     opts.Database,
		Branch:       opts.Branch,
	})
	if err != nil {
		return fmt.Errorf("list postgres roles: %w", err)
	}

	if err := ensureDefaultPostgresRole(ctx, psClient, opts, roles); err != nil {
		return err
	}

	var firstErr error
	for _, role := range roles {
		if !isStaleImportRole(role, currentUsername) {
			continue
		}
		err := psClient.PostgresRoles.Delete(ctx, &ps.DeletePostgresRoleRequest{
			Organization: opts.Org,
			Database:     opts.Database,
			Branch:       opts.Branch,
			RoleId:       role.ID,
			Successor:    postgresRoleName,
		})
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("delete stale import role %q: %w", role.Username, err)
		}
	}
	return firstErr
}

func ensureDefaultPostgresRole(ctx context.Context, psClient *ps.Client, opts ImportOptions, roles []*ps.PostgresRole) error {
	for _, role := range roles {
		if role != nil && isDefaultPostgresRole(role.Username) {
			return nil
		}
	}

	_, err := psClient.PostgresRoles.ResetDefaultRole(ctx, &ps.ResetDefaultRoleRequest{
		Organization: opts.Org,
		Database:     opts.Database,
		Branch:       opts.Branch,
	})
	if err != nil {
		return fmt.Errorf("ensure default postgres role: %w", err)
	}
	return nil
}

func isDefaultPostgresRole(username string) bool {
	return username == postgresRoleName || strings.HasPrefix(username, postgresRoleName+".")
}

const importRoleNamePrefix = "d1-import-"

func isImportRoleName(name string) bool {
	return strings.HasPrefix(name, importRoleNamePrefix)
}

func isStaleImportRole(role *ps.PostgresRole, currentUsername string) bool {
	if role == nil || role.Username == currentUsername {
		return false
	}
	// Only delete roles created by this import flow (d1-import-*). Do not touch
	// other ephemeral API roles (shell, manual admin roles, concurrent work).
	return isImportRoleName(role.Name)
}

func usernameFromDestURI(destURI string) (string, error) {
	cfg, err := postgres.ParseConnectionURI(destURI)
	if err != nil {
		return "", err
	}
	if cfg.User == "" {
		return "", fmt.Errorf("destination URI missing user")
	}
	return cfg.User, nil
}

func importTableNames(tables []TableSchema) []string {
	names := make([]string, 0, len(tables))
	for _, table := range tables {
		if IsORMMetadataTable(table.Name) {
			continue
		}
		names = append(names, table.Name)
	}
	return names
}

func existingPublicTables(ctx context.Context, destURI string, names []string) (map[string]struct{}, error) {
	existing := make(map[string]struct{})
	if len(names) == 0 {
		return existing, nil
	}

	db, err := OpenPostgres(destURI)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	placeholders := make([]string, len(names))
	args := make([]any, len(names))
	for i, name := range names {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = name
	}
	query := fmt.Sprintf(
		`SELECT table_name FROM information_schema.tables WHERE table_schema = '%s' AND table_name IN (%s)`,
		publicSchemaName,
		strings.Join(placeholders, ", "),
	)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list existing tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list existing tables: %w", err)
	}

	return existing, nil
}

func populatedLoadedTables(ctx context.Context, destURI string, loaded []string) ([]string, error) {
	if len(loaded) == 0 {
		return nil, nil
	}
	withRows, err := destTablesWithRows(ctx, destURI, loaded)
	if err != nil {
		return nil, err
	}
	return skipLoadedTablesForResume(loaded, withRows), nil
}

func skipLoadedTablesForResume(loaded []string, withRows map[string]struct{}) []string {
	populated := make([]string, 0, len(loaded))
	for _, table := range loaded {
		if _, ok := withRows[table]; ok {
			populated = append(populated, table)
		}
	}
	return populated
}

func destTablesWithRows(ctx context.Context, destURI string, tables []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	if len(tables) == 0 {
		return out, nil
	}

	db, err := OpenPostgres(destURI)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	for _, table := range tables {
		var hasRows bool
		query := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s LIMIT 1)`, postgres.QuoteIdentifier(table))
		if err := db.QueryRowContext(ctx, query).Scan(&hasRows); err != nil {
			return nil, fmt.Errorf("check rows in %s: %w", table, err)
		}
		if hasRows {
			out[table] = struct{}{}
		}
	}
	return out, nil
}

func conflictingImportTables(importNames []string, existing map[string]struct{}) []string {
	if len(existing) == 0 {
		return nil
	}
	conflicts := make([]string, 0, len(importNames))
	for _, name := range importNames {
		if _, found := existing[name]; found {
			conflicts = append(conflicts, name)
		}
	}
	return conflicts
}

func buildImportTablesSQL(inputPath string, tables []TableSchema) (string, error) {
	var coerceCtx *TypeCoercionContext
	if inputPath != "" {
		var err error
		coerceCtx, err = BuildTypeCoercionContext(inputPath, tables)
		if err != nil {
			return "", err
		}
	}

	tableByName := make(map[string]TableSchema, len(tables))
	for _, table := range tables {
		tableByName[table.Name] = table
	}

	var b strings.Builder
	for _, name := range topologicalLoadOrder(tables) {
		table, ok := tableByName[name]
		if !ok || IsORMMetadataTable(table.Name) {
			continue
		}
		b.WriteString(convertTableDDL(table, tables, coerceCtx))
		b.WriteString("\n\n")
	}
	return b.String(), nil
}
