package branch

import (
	"fmt"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/postgres"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ImportCompleteCmd returns the command for completing a PostgreSQL import.
func ImportCompleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		publication      string
		skipSeconds      int
		keepReplication  bool
		force            bool
		noCleanup        bool
		bufferMultiplier float64
		sampleDuration   time.Duration
	}

	cmd := &cobra.Command{
		Use:   "complete <database> <branch>",
		Short: "Complete a PostgreSQL import",
		Long: `Complete a PostgreSQL import by finalizing the migration.

This command performs the following steps:
1. Verifies all tables are replicating
2. Checks replication lag is minimal
3. Fast-forwards sequences to prevent conflicts
4. Disables and drops the subscription
5. Drops the publication on the source
6. Cleans up the replication role
7. Clears stored credentials

The sequence fast-forwarding samples sequence values over a short period,
calculates the rate of change, and advances destination sequences to
accommodate the specified skip time with a safety buffer.`,
		Example: `  # Complete an import
  pscale branch import complete mydb main

  # Complete with longer skip time for sequences
  pscale branch import complete mydb main --skip-seconds 120

  # Keep replication active (don't drop subscription)
  pscale branch import complete mydb main --keep-replication

  # Skip cleanup on error
  pscale branch import complete mydb main --no-cleanup`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

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

			// Get stored credentials
			creds, err := postgres.NewImportCredentials()
			if err != nil {
				return fmt.Errorf("failed to access credentials: %w", err)
			}

			info, err := creds.GetImportInfo(ch.Config.Organization, database, branch)
			if err != nil {
				return fmt.Errorf("no active import found for %s/%s", database, branch)
			}

			subName := info.SubscriptionName
			pubName := info.PublicationName
			if flags.publication != "" {
				pubName = flags.publication
			}

			ch.Printer.Printf("%s\n", printer.Bold("Import Completion"))
			ch.Printer.Printf("Database: %s/%s (branch: %s)\n", ch.Config.Organization, database, branch)
			ch.Printer.Printf("Subscription: %s\n", subName)
			ch.Printer.Printf("Publication: %s\n", pubName)

			// Two-stage confirmation
			if !flags.force {
				ch.Printer.Printf("\n%s This will:\n", printer.BoldYellow("Warning!"))
				ch.Printer.Printf("  - Fast-forward sequences (with %ds buffer)\n", flags.skipSeconds)
				if !flags.keepReplication {
					ch.Printer.Printf("  - Stop replication from source\n")
					ch.Printer.Printf("  - Drop the subscription on destination\n")
					ch.Printer.Printf("  - Drop the publication on source\n")
					ch.Printer.Printf("  - Clean up replication resources\n")
				}

				// First confirmation: type branch name
				if err := ch.Printer.ConfirmCommand(branch, "import complete", "import completion"); err != nil {
					return err
				}

				// Second confirmation: type "yes"
				var confirm string
				prompt := &survey.Input{
					Message: "Type 'yes' to confirm completion:",
				}
				if err := survey.AskOne(prompt, &confirm); err != nil {
					return err
				}
				if confirm != "yes" {
					return fmt.Errorf("import completion cancelled")
				}
			}

			// Try to retrieve existing role credentials
			username, password, host, err := creds.RetrieveRoleCredentials(ch.Config.Organization, database, branch)
			var dstCfg *postgres.Config

			if err == nil && username != "" {
				// Use existing role credentials
				ch.Printer.Printf("Using existing replication role\n")
				dstCfg = &postgres.Config{
					Host:     host,
					Port:     5432,
					User:     username,
					Password: password,
					Database: info.DBName,
					SSLMode:  "require",
					Options:  make(map[string]string),
				}
			} else {
				// Create temporary role (with postgres permissions to manage subscriptions)
				role, err := createTempRole(ctx, client, ch.Config.Organization, database, branch)
				if err != nil {
					return fmt.Errorf("failed to create temporary role: %w", err)
				}
				defer role.Cleanup(ctx, "postgres")

				dstCfg = &postgres.Config{
					Host:     role.Role.AccessHostURL,
					Port:     5432,
					User:     role.Role.Username,
					Password: role.Role.Password,
					Database: info.DBName,
					SSLMode:  "require",
					Options:  make(map[string]string),
				}
			}
			dst := postgres.BuildConnectionString(dstCfg)

			dstDB, err := postgres.OpenConnection(dst)
			if err != nil {
				return fmt.Errorf("failed to connect to destination: %w", err)
			}
			defer dstDB.Close()

			// Check subscription status
			end := ch.Printer.PrintProgress("Checking subscription status...")
			status, err := postgres.GetSubscriptionStatus(ctx, dstDB, subName)
			if err != nil {
				end()
				return fmt.Errorf("failed to get subscription status: %w", err)
			}
			end()

			// Check table states
			tableStates, err := postgres.GetTableReplicationStates(ctx, dstDB, subName)
			if err != nil {
				return fmt.Errorf("failed to get table states: %w", err)
			}

			notReady := 0
			for _, t := range tableStates {
				if t.State != "r" {
					notReady++
				}
			}

			if notReady > 0 && !flags.force {
				return fmt.Errorf("%d tables are not yet ready for replication. Wait for all tables to be ready or use --force", notReady)
			}

			ch.Printer.Printf("Subscription status: %s\n", printer.BoldGreen("ready"))
			ch.Printer.Printf("Tables ready: %d/%d\n", len(tableStates)-notReady, len(tableStates))

			// Connect to source for sequence sampling and cleanup
			src := info.SourceConnStr
			srcDB, err := postgres.OpenConnection(src)
			if err != nil {
				ch.Printer.Printf("%s Could not connect to source database: %v\n", printer.BoldYellow("[WARN]"), err)
				ch.Printer.Printf("  Sequence fast-forwarding and source cleanup will be skipped\n")
			} else {
				defer srcDB.Close()

				// Fast-forward sequences
				end = ch.Printer.PrintProgress(fmt.Sprintf("Fast-forwarding sequences (sampling for %s)...", flags.sampleDuration))
				result, err := postgres.FastForwardSequences(ctx, srcDB, dstDB, flags.sampleDuration, flags.bufferMultiplier, flags.skipSeconds)
				end()

				if err != nil {
					ch.Printer.Printf("%s Failed to fast-forward sequences: %v\n", printer.BoldYellow("[WARN]"), err)
				} else {
					ch.Printer.Printf("Sequences fast-forwarded: %d (skipped: %d)\n", result.TotalForwarded, result.TotalSkipped)
					if len(result.Sequences) > 0 && len(result.Sequences) <= 5 {
						for _, seq := range result.Sequences {
							ch.Printer.Printf("  %s.%s: %d -> %d (+%d)\n",
								seq.SchemaName, seq.SequenceName, seq.OldValue, seq.NewValue, seq.IncreasedBy)
						}
					}
				}
			}

			if !flags.keepReplication {
				// Disable subscription first
				end = ch.Printer.PrintProgress("Disabling subscription...")
				if err := postgres.DisableSubscription(ctx, dstDB, subName); err != nil {
					end()
					ch.Printer.Printf("%s Could not disable subscription: %v\n", printer.BoldYellow("[WARN]"), err)
					ch.Printer.Printf("  Will attempt to drop anyway...\n")
				} else {
					end()
					ch.Printer.Printf("Subscription disabled: %s\n", printer.BoldGreen("OK"))
				}

				// Drop subscription (use DROP ... CASCADE to force)
				end = ch.Printer.PrintProgress("Dropping subscription...")
				query := fmt.Sprintf("DROP SUBSCRIPTION IF EXISTS %s CASCADE", postgres.QuoteIdentifier(subName))
				_, err = dstDB.ExecContext(ctx, query)
				if err != nil {
					end()
					if !flags.noCleanup {
						return fmt.Errorf("failed to drop subscription: %w", err)
					}
					ch.Printer.Printf("%s Failed to drop subscription: %v\n", printer.BoldYellow("[WARN]"), err)
				} else {
					end()
					ch.Printer.Printf("Subscription dropped: %s\n", printer.BoldGreen("OK"))
				}

				// Drop replication slot on source
				if srcDB != nil {
					slotName := subName // Slot name matches subscription name
					end = ch.Printer.PrintProgress("Dropping replication slot on source...")
					if err := postgres.DropReplicationSlot(ctx, srcDB, slotName); err != nil {
						end()
						ch.Printer.Printf("%s Failed to drop replication slot: %v\n", printer.BoldYellow("[WARN]"), err)
					} else {
						end()
						ch.Printer.Printf("Replication slot dropped: %s\n", printer.BoldGreen("OK"))
					}
				}

				// Drop publication on source
				if srcDB != nil && pubName != "" {
					end = ch.Printer.PrintProgress("Dropping publication on source...")
					if err := postgres.DropPublication(ctx, srcDB, pubName, true); err != nil {
						end()
						ch.Printer.Printf("%s Failed to drop publication: %v\n", printer.BoldYellow("[WARN]"), err)
					} else {
						end()
						ch.Printer.Printf("Publication dropped: %s\n", printer.BoldGreen("OK"))
					}
				}

				// Delete the import replication role if we have the ID
				if info.RoleID != "" {
					end = ch.Printer.PrintProgress("Cleaning up replication role...")
					err := client.PostgresRoles.Delete(ctx, &ps.DeletePostgresRoleRequest{
						Organization: ch.Config.Organization,
						Database:     database,
						Branch:       branch,
						RoleId:       info.RoleID,
						Successor:    "postgres",
					})
					end()
					if err != nil {
						ch.Printer.Printf("%s Failed to delete replication role: %v\n", printer.BoldYellow("[WARN]"), err)
					} else {
						ch.Printer.Printf("Replication role cleaned up: %s\n", printer.BoldGreen("OK"))
					}
				}

				// Clear stored credentials
				if err := creds.ClearImportCredentials(ch.Config.Organization, database, branch); err != nil {
					ch.Printer.Printf("%s Failed to clear credentials: %v\n", printer.BoldYellow("[WARN]"), err)
				}
			} else {
				ch.Printer.Printf("\n%s Replication kept active (--keep-replication)\n", printer.BoldYellow("Note:"))
				ch.Printer.Printf("  Subscription: %s\n", status.Name)
				ch.Printer.Printf("  Publication: %s\n", status.PublicationName)
			}

			ch.Printer.Printf("\n%s Import completed successfully!\n", printer.BoldGreen("Success!"))
			ch.Printer.Printf("\nYour PlanetScale database is now ready to use.\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.publication, "publication", "", "Publication name (if not using stored value)")
	cmd.Flags().IntVar(&flags.skipSeconds, "skip-seconds", 60, "Seconds to buffer for sequence fast-forwarding")
	cmd.Flags().BoolVar(&flags.keepReplication, "keep-replication", false, "Keep replication active (don't drop subscription)")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Skip confirmation prompts")
	cmd.Flags().BoolVar(&flags.noCleanup, "no-cleanup", false, "Continue on cleanup errors")
	cmd.Flags().Float64Var(&flags.bufferMultiplier, "buffer-multiplier", 2.0, "Safety multiplier for sequence fast-forwarding")
	cmd.Flags().DurationVar(&flags.sampleDuration, "sample-duration", 10*time.Second, "Duration to sample sequences")

	return cmd
}
