package branch

import (
	"context"
	"encoding/json"
	"fmt"

	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/postgres"
	"github.com/planetscale/cli/internal/printer"
)

// ImportListOutput represents the JSON output for import list.
type ImportListOutput struct {
	Imports []ImportListItem `json:"imports"`
}

// ImportListItem represents a single import in the list.
type ImportListItem struct {
	Database     string `json:"database"`
	Branch       string `json:"branch"`
	Subscription string `json:"subscription"`
	Publication  string `json:"publication"`
	Enabled      bool   `json:"enabled"`
	TablesTotal  int    `json:"tables_total"`
	TablesReady  int    `json:"tables_ready"`
	Status       string `json:"status"`
}

func ImportListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		status          string
		showCredentials bool
		format          printer.Format
	}

	cmd := &cobra.Command{
		Use:   "list <database> <branch>",
		Short: "List active PostgreSQL imports for a branch",
		Long: `List active PostgreSQL imports for a specific branch.

Shows all active subscriptions that were created by the import process,
along with their current status and progress.`,
		Example: `  # List imports for a branch
  pscale branch import list mydb main

  # Output as JSON
  pscale branch import list mydb main --format json

  # Filter by status
  pscale branch import list mydb main --status active`,
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
					return fmt.Errorf("database %s does not exist", printer.BoldBlue(database))
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

			imports := importItems(ctx, client, creds, ch.Config.Organization, database, branch, flags.status)

			if flags.format == printer.JSON {
				return formatImportsJSON(ch, imports)
			}
			return printImports(ch, creds, database, branch, imports, flags.showCredentials)
		},
	}

	cmd.Flags().StringVar(&flags.status, "status", "", "Filter by status (active, syncing, ready, disabled)")
	cmd.Flags().BoolVar(&flags.showCredentials, "show-credentials", false, "Show source connection info (redacted)")
	cmd.Flags().Var(printer.NewFormatValue(printer.Human, &flags.format), "format", "Output format (human, json)")

	return cmd
}

func importItems(ctx context.Context, client *ps.Client, creds *postgres.ImportCredentials, org, database, branch, statusFilter string) []ImportListItem {
	var imports []ImportListItem

	subs, err := creds.ListStoredSubscriptions(org, database, branch)
	if err != nil || len(subs) == 0 {
		return imports
	}

	for _, subName := range subs {
		info, err := creds.GetImportInfoForSubscription(org, database, branch, subName)
		if err != nil {
			continue
		}

		item := ImportListItem{
			Database:     database,
			Branch:       branch,
			Subscription: info.SubscriptionName,
			Publication:  info.PublicationName,
			Status:       "unknown",
		}

		fetchImportItemStatus(ctx, client, org, database, branch, info, &item)

		if statusFilter != "" && statusFilter != item.Status {
			continue
		}

		imports = append(imports, item)
	}

	return imports
}

func fetchImportItemStatus(ctx context.Context, client *ps.Client, org, database, branch string, info *postgres.ImportInfo, item *ImportListItem) {
	role, err := createTempRole(ctx, client, org, database, branch)
	if err != nil {
		return
	}
	defer role.Cleanup(ctx, "postgres")

	cfg := &postgres.Config{
		Host:     role.Role.AccessHostURL,
		Port:     5432,
		User:     role.Role.Username,
		Password: role.Role.Password,
		Database: info.DBName,
		SSLMode:  "require",
		Options:  make(map[string]string),
	}

	db, err := postgres.OpenConnection(postgres.BuildConnectionString(cfg))
	if err != nil {
		return
	}
	defer db.Close()

	status, err := postgres.GetSubscriptionStatus(ctx, db, info.SubscriptionName)
	if err == nil {
		item.Enabled = status.Enabled
		if status.Enabled {
			item.Status = "active"
		} else {
			item.Status = "disabled"
		}
	}

	tables, err := postgres.GetTableReplicationStates(ctx, db, info.SubscriptionName)
	if err == nil {
		item.TablesTotal = len(tables)
		for _, t := range tables {
			if t.State == "r" {
				item.TablesReady++
			}
		}
		if item.TablesReady == item.TablesTotal && item.TablesTotal > 0 {
			item.Status = "ready"
		} else if item.TablesReady < item.TablesTotal {
			item.Status = "syncing"
		}
	}
}

func formatImportsJSON(ch *cmdutil.Helper, imports []ImportListItem) error {
	output := ImportListOutput{Imports: imports}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	ch.Printer.Println(string(data))
	return nil
}

func printImports(ch *cmdutil.Helper, creds *postgres.ImportCredentials, database, branch string, imports []ImportListItem, showCreds bool) error {
	if len(imports) == 0 {
		ch.Printer.Printf("No active imports found for %s/%s\n",
			printer.BoldBlue(database), printer.BoldBlue(branch))
		return nil
	}

	ch.Printer.Printf("%s for %s/%s\n\n", printer.Bold("Active Imports"),
		printer.BoldBlue(database), printer.BoldBlue(branch))

	for _, imp := range imports {
		ch.Printer.Printf("Subscription: %s\n", printer.BoldBlue(imp.Subscription))
		ch.Printer.Printf("  Publication: %s\n", imp.Publication)

		statusColor := printer.BoldYellow
		switch imp.Status {
		case "ready":
			statusColor = printer.BoldGreen
		case "active", "syncing":
			statusColor = printer.BoldBlue
		case "disabled":
			statusColor = printer.BoldRed
		}
		ch.Printer.Printf("  Status: %s\n", statusColor(imp.Status))

		if imp.TablesTotal > 0 {
			ch.Printer.Printf("  Tables: %d/%d ready\n", imp.TablesReady, imp.TablesTotal)
		}

		if showCreds {
			info, _ := creds.GetImportInfoForSubscription(ch.Config.Organization, database, branch, imp.Subscription)
			if info != nil {
				ch.Printer.Printf("  Source: %s\n", postgres.RedactPassword(info.SourceConnStr))
			}
		}

		ch.Printer.Printf("\n")
	}

	return nil
}
