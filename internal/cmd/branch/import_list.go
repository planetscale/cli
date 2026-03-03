package branch

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/postgres"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
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

// ImportListCmd returns the command for listing PostgreSQL imports.
func ImportListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		status          string
		showCredentials bool
		format          printer.Format
	}

	cmd := &cobra.Command{
		Use:   "list <database> [branch]",
		Short: "List active PostgreSQL imports",
		Long: `List active PostgreSQL imports for a database or specific branch.

Shows all active subscriptions that were created by the import process,
along with their current status and progress.`,
		Example: `  # List all imports for a database
  pscale branch import list mydb

  # List import for a specific branch
  pscale branch import list mydb main

  # Output as JSON
  pscale branch import list mydb --format json

  # Filter by status
  pscale branch import list mydb --status active`,
		Args: cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			var specificBranch string
			if len(args) > 1 {
				specificBranch = args[1]
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			// Check if this is a PostgreSQL database
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

			// Get list of branches
			var branches []string
			if specificBranch != "" {
				branches = []string{specificBranch}
			} else {
				// List all branches
				branchList, err := client.PostgresBranches.List(ctx, &ps.ListPostgresBranchesRequest{
					Organization: ch.Config.Organization,
					Database:     database,
				})
				if err != nil {
					return fmt.Errorf("failed to list branches: %w", err)
				}
				for _, b := range branchList {
					branches = append(branches, b.Name)
				}
			}

			// Check credentials for each branch
			creds, err := postgres.NewImportCredentials()
			if err != nil {
				return fmt.Errorf("failed to access credentials: %w", err)
			}

			var imports []ImportListItem

			for _, branch := range branches {
				// Check if this branch has import credentials
				if !creds.HasImportCredentials(ch.Config.Organization, database, branch) {
					continue
				}

				info, err := creds.GetImportInfo(ch.Config.Organization, database, branch)
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

				// Try to get detailed status
				role, err := createTempRole(ctx, client, ch.Config.Organization, database, branch)
				if err == nil {
					dstCfg := &postgres.Config{
						Host:     role.Role.AccessHostURL,
						Port:     5432,
						User:     role.Role.Username,
						Password: role.Role.Password,
						Database: info.DBName,
						SSLMode:  "require",
						Options:  make(map[string]string),
					}
					dst := postgres.BuildConnectionString(dstCfg)

					dstDB, err := postgres.OpenConnection(dst)
					if err == nil {
						// Get subscription status
						status, err := postgres.GetSubscriptionStatus(ctx, dstDB, info.SubscriptionName)
						if err == nil {
							item.Enabled = status.Enabled
							if status.Enabled {
								item.Status = "active"
							} else {
								item.Status = "disabled"
							}
						}

						// Get table states
						tableStates, err := postgres.GetTableReplicationStates(ctx, dstDB, info.SubscriptionName)
						if err == nil {
							item.TablesTotal = len(tableStates)
							for _, t := range tableStates {
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

						dstDB.Close()
					}
					role.Cleanup(ctx, "postgres")
				}

				// Apply status filter
				if flags.status != "" && flags.status != item.Status {
					continue
				}

				imports = append(imports, item)
			}

			// Output
			if flags.format == printer.JSON {
				output := ImportListOutput{Imports: imports}
				data, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return err
				}
				ch.Printer.Println(string(data))
				return nil
			}

			// Human-readable output
			if len(imports) == 0 {
				if specificBranch != "" {
					ch.Printer.Printf("No active imports found for branch %s in database %s\n",
						printer.BoldBlue(specificBranch), printer.BoldBlue(database))
				} else {
					ch.Printer.Printf("No active imports found for database %s\n", printer.BoldBlue(database))
				}
				return nil
			}

			ch.Printer.Printf("%s\n\n", printer.Bold("Active Imports"))

			for _, imp := range imports {
				ch.Printer.Printf("%s/%s:\n", printer.BoldBlue(imp.Database), printer.BoldBlue(imp.Branch))
				ch.Printer.Printf("  Subscription: %s\n", imp.Subscription)
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

				if flags.showCredentials {
					info, _ := creds.GetImportInfo(ch.Config.Organization, database, imp.Branch)
					if info != nil {
						ch.Printer.Printf("  Source: %s\n", postgres.RedactPassword(info.SourceConnStr))
					}
				}

				ch.Printer.Printf("\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.status, "status", "", "Filter by status (active, syncing, ready, disabled)")
	cmd.Flags().BoolVar(&flags.showCredentials, "show-credentials", false, "Show source connection info (redacted)")
	cmd.Flags().Var(printer.NewFormatValue(printer.Human, &flags.format), "format", "Output format (human, json)")

	return cmd
}
