package branch

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/postgres"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ImportCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		subscription    string
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

			subName, err := resolveSubscriptionForCancel(creds, ch.Config.Organization, database, branch, flags.subscription)
			if err != nil {
				return err
			}

			info, err := creds.GetImportInfoForSubscription(ch.Config.Organization, database, branch, subName)
			if err != nil {
				return fmt.Errorf("failed to retrieve import info: %w", err)
			}

			pubName := info.PublicationName
			if flags.publication != "" {
				pubName = flags.publication
			}

			if err := printCancelSummary(ch, database, branch, subName, pubName, flags.dropSchema, flags.force); err != nil {
				return err
			}

			var errs []string

			dstDB, cleanup, err := connectForCleanup(ctx, client, ch.Config.Organization, database, branch, info, &errs)
			if cleanup != nil {
				defer cleanup()
			}

			if dstDB != nil {
				defer dstDB.Close()
				cleanupDestination(ctx, ch, dstDB, subName, flags.dropSchema, &errs)
			}

			cleanupSource(ctx, ch, info.SourceConnStr, pubName, subName, &errs)
			cleanupRoleAndCredentials(ctx, ch, client, creds, ch.Config.Organization, database, branch, subName, info.RoleID, flags.keepCredentials, &errs)

			printCancelResult(ch, errs)
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.subscription, "subscription", "", "Subscription name (for selecting specific import)")
	cmd.Flags().StringVar(&flags.publication, "publication", "", "Publication name (if not using stored value)")
	cmd.Flags().BoolVar(&flags.dropSchema, "drop-schema", false, "Drop imported tables and schema")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&flags.keepCredentials, "keep-credentials", false, "Keep stored credentials")

	return cmd
}

func resolveSubscriptionForCancel(creds *postgres.ImportCredentials, org, db, branch, flagSub string) (string, error) {
	if flagSub != "" {
		return flagSub, nil
	}

	subs, err := creds.ListStoredSubscriptions(org, db, branch)
	if err != nil {
		return "", fmt.Errorf("failed to list imports: %w", err)
	}

	if len(subs) == 0 {
		return "", fmt.Errorf("no active imports found for %s/%s", db, branch)
	}

	if len(subs) > 1 {
		var selected string
		prompt := &survey.Select{
			Message: "Multiple imports found. Which one would you like to cancel?",
			Options: subs,
		}
		if err := survey.AskOne(prompt, &selected); err != nil {
			return "", err
		}
		return selected, nil
	}

	return subs[0], nil
}

func printCancelSummary(ch *cmdutil.Helper, database, branch, subName, pubName string, dropSchema, force bool) error {
	ch.Printer.Printf("%s\n", printer.Bold("Cancel Import"))
	ch.Printer.Printf("Database: %s/%s (branch: %s)\n", ch.Config.Organization, database, branch)
	ch.Printer.Printf("Subscription: %s\n", subName)
	ch.Printer.Printf("Publication: %s\n", pubName)

	if force {
		return nil
	}

	ch.Printer.Printf("\n%s This will:\n", printer.BoldYellow("Warning!"))
	ch.Printer.Printf("  - Stop replication from source\n")
	ch.Printer.Printf("  - Drop the subscription on destination\n")
	ch.Printer.Printf("  - Drop the publication on source\n")
	if dropSchema {
		ch.Printer.Printf("  - Drop all imported tables and schema\n")
	}
	ch.Printer.Printf("  - Clean up replication resources\n")

	return ch.Printer.ConfirmCommand(branch, "import cancel", "import cancellation")
}

func connectForCleanup(ctx context.Context, client *ps.Client, org, database, branch string, info *postgres.ImportInfo, errs *[]string) (*sql.DB, func(), error) {
	var cfg *postgres.Config
	var cleanup func()

	if info.RoleUsername != "" && info.RolePassword != "" && info.RoleHost != "" {
		cfg = &postgres.Config{
			Host:     info.RoleHost,
			Port:     5432,
			User:     info.RoleUsername,
			Password: info.RolePassword,
			Database: info.DBName,
			SSLMode:  "require",
			Options:  make(map[string]string),
		}
	} else {
		role, err := createTempRole(ctx, client, org, database, branch)
		if err != nil {
			*errs = append(*errs, fmt.Sprintf("create temp role: %v", err))
			return nil, nil, err
		}
		cfg = &postgres.Config{
			Host:     role.Role.AccessHostURL,
			Port:     5432,
			User:     role.Role.Username,
			Password: role.Role.Password,
			Database: info.DBName,
			SSLMode:  "require",
			Options:  make(map[string]string),
		}
		cleanup = func() { role.Cleanup(ctx, "postgres") }
	}

	connStr := postgres.BuildConnectionString(cfg)
	db, err := postgres.OpenConnection(connStr)
	if err != nil {
		*errs = append(*errs, fmt.Sprintf("connect to destination: %v", err))
		return nil, cleanup, err
	}

	return db, cleanup, nil
}

func cleanupDestination(ctx context.Context, ch *cmdutil.Helper, db *sql.DB, subName string, dropSchema bool, errs *[]string) {
	end := ch.Printer.PrintProgress("Disabling subscription...")
	if err := postgres.DisableSubscription(ctx, db, subName); err != nil {
		end()
		*errs = append(*errs, fmt.Sprintf("disable subscription: %v", err))
		ch.Printer.Printf("%s Failed to disable subscription: %v\n", printer.BoldYellow("[WARN]"), err)
	} else {
		end()
		ch.Printer.Printf("Subscription disabled: %s\n", printer.BoldGreen("OK"))
	}

	end = ch.Printer.PrintProgress("Dropping subscription...")
	if err := postgres.DropSubscription(ctx, db, subName, true); err != nil {
		end()
		*errs = append(*errs, fmt.Sprintf("drop subscription: %v", err))
		ch.Printer.Printf("%s Failed to drop subscription: %v\n", printer.BoldYellow("[WARN]"), err)
	} else {
		end()
		ch.Printer.Printf("Subscription dropped: %s\n", printer.BoldGreen("OK"))
	}

	if !dropSchema {
		return
	}

	end = ch.Printer.PrintProgress("Dropping imported schema...")
	_, err := db.ExecContext(ctx, `
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
		*errs = append(*errs, fmt.Sprintf("drop schema: %v", err))
		ch.Printer.Printf("%s Failed to drop schema: %v\n", printer.BoldYellow("[WARN]"), err)
	} else {
		ch.Printer.Printf("Schema dropped: %s\n", printer.BoldGreen("OK"))
	}
}

func cleanupSource(ctx context.Context, ch *cmdutil.Helper, connStr, pubName, subName string, errs *[]string) {
	if connStr == "" {
		return
	}

	db, err := postgres.OpenConnection(connStr)
	if err != nil {
		ch.Printer.Printf("%s Could not connect to source for cleanup: %v\n", printer.BoldYellow("[WARN]"), err)
		return
	}
	defer db.Close()

	if pubName != "" {
		end := ch.Printer.PrintProgress("Dropping publication on source...")
		if err := postgres.DropPublication(ctx, db, pubName, true); err != nil {
			end()
			*errs = append(*errs, fmt.Sprintf("drop publication: %v", err))
			ch.Printer.Printf("%s Failed to drop publication: %v\n", printer.BoldYellow("[WARN]"), err)
		} else {
			end()
			ch.Printer.Printf("Publication dropped: %s\n", printer.BoldGreen("OK"))
		}
	}

	slotName := subName
	end := ch.Printer.PrintProgress("Dropping replication slot on source...")
	if err := postgres.DropReplicationSlot(ctx, db, slotName); err != nil {
		end()
		ch.Printer.Printf("Replication slot: %s (may already be dropped)\n", printer.BoldYellow("skipped"))
	} else {
		end()
		ch.Printer.Printf("Replication slot dropped: %s\n", printer.BoldGreen("OK"))
	}
}

func cleanupRoleAndCredentials(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, creds *postgres.ImportCredentials, org, database, branch, subName, roleID string, keepCreds bool, errs *[]string) {
	if roleID != "" {
		end := ch.Printer.PrintProgress("Cleaning up replication role...")
		err := client.PostgresRoles.Delete(ctx, &ps.DeletePostgresRoleRequest{
			Organization: org,
			Database:     database,
			Branch:       branch,
			RoleId:       roleID,
			Successor:    "postgres",
		})
		end()
		if err != nil {
			*errs = append(*errs, fmt.Sprintf("delete role: %v", err))
			ch.Printer.Printf("%s Failed to delete replication role: %v\n", printer.BoldYellow("[WARN]"), err)
		} else {
			ch.Printer.Printf("Replication role cleaned up: %s\n", printer.BoldGreen("OK"))
		}
	}

	if keepCreds {
		return
	}

	if err := creds.ClearSubscriptionCredentials(org, database, branch, subName); err != nil {
		*errs = append(*errs, fmt.Sprintf("clear credentials: %v", err))
		ch.Printer.Printf("%s Failed to clear credentials: %v\n", printer.BoldYellow("[WARN]"), err)
	} else {
		ch.Printer.Printf("Credentials cleared: %s\n", printer.BoldGreen("OK"))
	}
}

func printCancelResult(ch *cmdutil.Helper, errs []string) {
	ch.Printer.Printf("\n")
	if len(errs) > 0 {
		ch.Printer.Printf("%s Import cancelled with some warnings.\n", printer.BoldYellow("Done!"))
		ch.Printer.Printf("Some cleanup steps failed (see warnings above).\n")
	} else {
		ch.Printer.Printf("%s Import cancelled and cleaned up successfully.\n", printer.BoldGreen("Success!"))
	}
}
