package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/postgres"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ImportCancelCmd returns the command for cancelling a PostgreSQL import.
func ImportCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		publication     string
		dropSchema      bool
		force           bool
		keepCredentials bool
	}

	cmd := &cobra.Command{
		Use:   "cancel <database> <branch>",
		Short: "Cancel a PostgreSQL import",
		Long: `Cancel an active PostgreSQL import and clean up resources.

This command performs graceful cleanup:
1. Disables the subscription on the destination
2. Drops the subscription
3. Drops the publication on the source (if accessible)
4. Drops the replication slot (if accessible)
5. Optionally drops the imported schema
6. Cleans up the replication role
7. Clears stored credentials

The command continues even if some cleanup steps fail, ensuring
as much cleanup as possible is performed.`,
		Example: `  # Cancel an import
  pscale branch import cancel mydb main

  # Cancel and drop imported schema
  pscale branch import cancel mydb main --drop-schema

  # Force cancel without confirmation
  pscale branch import cancel mydb main --force`,
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

			ch.Printer.Printf("%s\n", printer.Bold("Cancel Import"))
			ch.Printer.Printf("Database: %s/%s (branch: %s)\n", ch.Config.Organization, database, branch)
			ch.Printer.Printf("Subscription: %s\n", subName)
			ch.Printer.Printf("Publication: %s\n", pubName)

			// Confirmation
			if !flags.force {
				ch.Printer.Printf("\n%s This will:\n", printer.BoldYellow("Warning!"))
				ch.Printer.Printf("  - Stop replication from source\n")
				ch.Printer.Printf("  - Drop the subscription on destination\n")
				ch.Printer.Printf("  - Drop the publication on source\n")
				if flags.dropSchema {
					ch.Printer.Printf("  - Drop all imported tables and schema\n")
				}
				ch.Printer.Printf("  - Clean up replication resources\n")

				if err := ch.Printer.ConfirmCommand(branch, "import cancel", "import cancellation"); err != nil {
					return err
				}
			}

			// Track cleanup errors but continue
			var cleanupErrors []string

			// Try to retrieve existing role credentials first
			username, password, host, err := creds.RetrieveRoleCredentials(ch.Config.Organization, database, branch)
			var dstCfg *postgres.Config

			if err == nil && username != "" {
				// Use existing role credentials
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
				// Create temporary role for destination access
				role, err := createTempRole(ctx, client, ch.Config.Organization, database, branch)
				if err != nil {
					cleanupErrors = append(cleanupErrors, fmt.Sprintf("create temp role: %v", err))
					dstCfg = nil
				} else {
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
			}

			var dst string
			if dstCfg != nil {
				dst = postgres.BuildConnectionString(dstCfg)
			}

			if dst != "" {
				dstDB, err := postgres.OpenConnection(dst)
				if err != nil {
					cleanupErrors = append(cleanupErrors, fmt.Sprintf("connect to destination: %v", err))
				} else {
					defer dstDB.Close()

					// Disable subscription
					end := ch.Printer.PrintProgress("Disabling subscription...")
					if err := postgres.DisableSubscription(ctx, dstDB, subName); err != nil {
						end()
						cleanupErrors = append(cleanupErrors, fmt.Sprintf("disable subscription: %v", err))
						ch.Printer.Printf("%s Failed to disable subscription: %v\n", printer.BoldYellow("[WARN]"), err)
					} else {
						end()
						ch.Printer.Printf("Subscription disabled: %s\n", printer.BoldGreen("OK"))
					}

					// Drop subscription
					end = ch.Printer.PrintProgress("Dropping subscription...")
					if err := postgres.DropSubscription(ctx, dstDB, subName, true); err != nil {
						end()
						cleanupErrors = append(cleanupErrors, fmt.Sprintf("drop subscription: %v", err))
						ch.Printer.Printf("%s Failed to drop subscription: %v\n", printer.BoldYellow("[WARN]"), err)
					} else {
						end()
						ch.Printer.Printf("Subscription dropped: %s\n", printer.BoldGreen("OK"))
					}

					// Drop schema if requested
					if flags.dropSchema {
						end = ch.Printer.PrintProgress("Dropping imported schema...")
						// Drop all user tables in the public schema
						_, err := dstDB.ExecContext(ctx, `
							DO $$
							DECLARE
								r RECORD;
							BEGIN
								FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public')
								LOOP
									EXECUTE 'DROP TABLE IF EXISTS public.' || quote_ident(r.tablename) || ' CASCADE';
								END LOOP;
							END $$;
						`)
						end()
						if err != nil {
							cleanupErrors = append(cleanupErrors, fmt.Sprintf("drop schema: %v", err))
							ch.Printer.Printf("%s Failed to drop schema: %v\n", printer.BoldYellow("[WARN]"), err)
						} else {
							ch.Printer.Printf("Schema dropped: %s\n", printer.BoldGreen("OK"))
						}
					}
				}
			}

			// Try to clean up source resources
			src := info.SourceConnStr
			if src != "" {
				srcDB, err := postgres.OpenConnection(src)
				if err != nil {
					ch.Printer.Printf("%s Could not connect to source for cleanup: %v\n", printer.BoldYellow("[WARN]"), err)
				} else {
					defer srcDB.Close()

					// Drop publication
					if pubName != "" {
						end := ch.Printer.PrintProgress("Dropping publication on source...")
						if err := postgres.DropPublication(ctx, srcDB, pubName, true); err != nil {
							end()
							cleanupErrors = append(cleanupErrors, fmt.Sprintf("drop publication: %v", err))
							ch.Printer.Printf("%s Failed to drop publication: %v\n", printer.BoldYellow("[WARN]"), err)
						} else {
							end()
							ch.Printer.Printf("Publication dropped: %s\n", printer.BoldGreen("OK"))
						}
					}

					// Try to drop replication slot (may already be gone with subscription)
					slotName := subName // Subscription name is usually the slot name
					end := ch.Printer.PrintProgress("Dropping replication slot on source...")
					if err := postgres.DropReplicationSlot(ctx, srcDB, slotName); err != nil {
						end()
						// This is expected to fail if the subscription drop already removed it
						ch.Printer.Printf("Replication slot: %s (may already be dropped)\n", printer.BoldYellow("skipped"))
					} else {
						end()
						ch.Printer.Printf("Replication slot dropped: %s\n", printer.BoldGreen("OK"))
					}
				}
			}

			// Delete the import replication role if we have the ID
			if info.RoleID != "" {
				end := ch.Printer.PrintProgress("Cleaning up replication role...")
				err := client.PostgresRoles.Delete(ctx, &ps.DeletePostgresRoleRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Branch:       branch,
					RoleId:       info.RoleID,
					Successor:    "postgres",
				})
				end()
				if err != nil {
					cleanupErrors = append(cleanupErrors, fmt.Sprintf("delete role: %v", err))
					ch.Printer.Printf("%s Failed to delete replication role: %v\n", printer.BoldYellow("[WARN]"), err)
				} else {
					ch.Printer.Printf("Replication role cleaned up: %s\n", printer.BoldGreen("OK"))
				}
			}

			// Clear stored credentials
			if !flags.keepCredentials {
				if err := creds.ClearImportCredentials(ch.Config.Organization, database, branch); err != nil {
					cleanupErrors = append(cleanupErrors, fmt.Sprintf("clear credentials: %v", err))
					ch.Printer.Printf("%s Failed to clear credentials: %v\n", printer.BoldYellow("[WARN]"), err)
				} else {
					ch.Printer.Printf("Credentials cleared: %s\n", printer.BoldGreen("OK"))
				}
			}

			ch.Printer.Printf("\n")
			if len(cleanupErrors) > 0 {
				ch.Printer.Printf("%s Import cancelled with some warnings.\n", printer.BoldYellow("Done!"))
				ch.Printer.Printf("Some cleanup steps failed (see warnings above).\n")
			} else {
				ch.Printer.Printf("%s Import cancelled and cleaned up successfully.\n", printer.BoldGreen("Success!"))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.publication, "publication", "", "Publication name (if not using stored value)")
	cmd.Flags().BoolVar(&flags.dropSchema, "drop-schema", false, "Drop imported tables and schema")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&flags.keepCredentials, "keep-credentials", false, "Keep stored credentials")

	return cmd
}
