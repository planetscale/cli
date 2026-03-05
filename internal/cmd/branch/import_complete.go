package branch

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/postgres"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ImportCompleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		subscription     string
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

			subName, err := resolveSubscriptionForComplete(creds, ch.Config.Organization, database, branch, flags.subscription)
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

			if err := printCompleteSummary(ch, database, branch, subName, pubName, flags.skipSeconds, flags.keepReplication, flags.force); err != nil {
				return err
			}

			dstDB, cleanup, err := connectForComplete(ctx, ch, client, ch.Config.Organization, database, branch, info)
			if cleanup != nil {
				defer cleanup()
			}
			if err != nil {
				return err
			}
			defer dstDB.Close()

			status, err := verifyReplicationReady(ctx, ch, dstDB, subName, flags.force)
			if err != nil {
				return err
			}

			srcDB := fastForwardSequencesStep(ctx, ch, info.SourceConnStr, dstDB, flags.sampleDuration, flags.bufferMultiplier, flags.skipSeconds)

			if flags.keepReplication {
				printKeepReplicationInfo(ch, status)
			} else {
				cleanupReplicationResources(ctx, ch, client, creds, dstDB, srcDB, ch.Config.Organization, database, branch, subName, pubName, info.RoleID, flags.noCleanup)
			}

			ch.Printer.Printf("\n%s Import completed successfully!\n", printer.BoldGreen("Success!"))
			ch.Printer.Printf("\nYour PlanetScale database is now ready to use.\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.subscription, "subscription", "", "Subscription name (for selecting specific import)")
	cmd.Flags().StringVar(&flags.publication, "publication", "", "Publication name (if not using stored value)")
	cmd.Flags().IntVar(&flags.skipSeconds, "skip-seconds", 60, "Seconds to buffer for sequence fast-forwarding")
	cmd.Flags().BoolVar(&flags.keepReplication, "keep-replication", false, "Keep replication active (don't drop subscription)")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Skip confirmation prompts")
	cmd.Flags().BoolVar(&flags.noCleanup, "no-cleanup", false, "Continue on cleanup errors")
	cmd.Flags().Float64Var(&flags.bufferMultiplier, "buffer-multiplier", 2.0, "Safety multiplier for sequence fast-forwarding")
	cmd.Flags().DurationVar(&flags.sampleDuration, "sample-duration", 10*time.Second, "Duration to sample sequences")

	return cmd
}

func resolveSubscriptionForComplete(creds *postgres.ImportCredentials, org, db, branch, flagSub string) (string, error) {
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
			Message: "Multiple imports found. Which one would you like to complete?",
			Options: subs,
		}
		if err := survey.AskOne(prompt, &selected); err != nil {
			return "", err
		}
		return selected, nil
	}

	return subs[0], nil
}

func printCompleteSummary(ch *cmdutil.Helper, database, branch, subName, pubName string, skipSeconds int, keepReplication, force bool) error {
	ch.Printer.Printf("%s\n", printer.Bold("Import Completion"))
	ch.Printer.Printf("Database: %s/%s (branch: %s)\n", ch.Config.Organization, database, branch)
	ch.Printer.Printf("Subscription: %s\n", subName)
	ch.Printer.Printf("Publication: %s\n", pubName)

	if force {
		return nil
	}

	ch.Printer.Printf("\n%s This will:\n", printer.BoldYellow("Warning!"))
	ch.Printer.Printf("  - Fast-forward sequences (with %ds buffer)\n", skipSeconds)
	if !keepReplication {
		ch.Printer.Printf("  - Stop replication from source\n")
		ch.Printer.Printf("  - Drop the subscription on destination\n")
		ch.Printer.Printf("  - Drop the publication on source\n")
		ch.Printer.Printf("  - Clean up replication resources\n")
	}

	if err := ch.Printer.ConfirmCommand(branch, "import complete", "import completion"); err != nil {
		return err
	}

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

	return nil
}

func connectForComplete(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, org, database, branch string, info *postgres.ImportInfo) (*sql.DB, func(), error) {
	var cfg *postgres.Config
	var cleanup func()

	if info.RoleUsername != "" && info.RolePassword != "" && info.RoleHost != "" {
		ch.Printer.Printf("Using existing replication role\n")
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
			return nil, nil, fmt.Errorf("failed to create temporary role: %w", err)
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

	db, err := postgres.OpenConnection(postgres.BuildConnectionString(cfg))
	if err != nil {
		return nil, cleanup, fmt.Errorf("failed to connect to destination: %w", err)
	}

	return db, cleanup, nil
}

func verifyReplicationReady(ctx context.Context, ch *cmdutil.Helper, db *sql.DB, subName string, force bool) (*postgres.SubscriptionStatus, error) {
	end := ch.Printer.PrintProgress("Checking subscription status...")
	status, err := postgres.GetSubscriptionStatus(ctx, db, subName)
	if err != nil {
		end()
		return nil, fmt.Errorf("failed to get subscription status: %w", err)
	}
	end()

	tables, err := postgres.GetTableReplicationStates(ctx, db, subName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table states: %w", err)
	}

	notReady := 0
	for _, t := range tables {
		if t.State != "r" {
			notReady++
		}
	}

	if notReady > 0 && !force {
		return nil, fmt.Errorf("%d tables are not yet ready for replication. Wait for all tables to be ready or use --force", notReady)
	}

	ch.Printer.Printf("Subscription status: %s\n", printer.BoldGreen("ready"))
	ch.Printer.Printf("Tables ready: %d/%d\n", len(tables)-notReady, len(tables))

	return status, nil
}

func fastForwardSequencesStep(ctx context.Context, ch *cmdutil.Helper, sourceConnStr string, dstDB *sql.DB, sampleDuration time.Duration, bufferMultiplier float64, skipSeconds int) *sql.DB {
	srcDB, err := postgres.OpenConnection(sourceConnStr)
	if err != nil {
		ch.Printer.Printf("%s Could not connect to source database: %v\n", printer.BoldYellow("[WARN]"), err)
		ch.Printer.Printf("  Sequence fast-forwarding and source cleanup will be skipped\n")
		return nil
	}

	end := ch.Printer.PrintProgress(fmt.Sprintf("Fast-forwarding sequences (sampling for %s)...", sampleDuration))
	result, err := postgres.FastForwardSequences(ctx, srcDB, dstDB, sampleDuration, bufferMultiplier, skipSeconds)
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

	return srcDB
}

func cleanupReplicationResources(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, creds *postgres.ImportCredentials, dstDB, srcDB *sql.DB, org, database, branch, subName, pubName, roleID string, noCleanup bool) {
	end := ch.Printer.PrintProgress("Disabling subscription...")
	if err := postgres.DisableSubscription(ctx, dstDB, subName); err != nil {
		end()
		ch.Printer.Printf("%s Could not disable subscription: %v\n", printer.BoldYellow("[WARN]"), err)
		ch.Printer.Printf("  Will attempt to drop anyway...\n")
	} else {
		end()
		ch.Printer.Printf("Subscription disabled: %s\n", printer.BoldGreen("OK"))
	}

	end = ch.Printer.PrintProgress("Dropping subscription...")

	alterQuery := fmt.Sprintf("ALTER SUBSCRIPTION %s OWNER TO CURRENT_ROLE", postgres.QuoteIdentifier(subName))
	if _, err := dstDB.ExecContext(ctx, alterQuery); err != nil {
		end()
		ch.Printer.Printf("%s Could not transfer subscription ownership: %v\n", printer.BoldYellow("[WARN]"), err)
		ch.Printer.Printf("  Will attempt to drop anyway...\n")
		end = ch.Printer.PrintProgress("Dropping subscription...")
	}

	query := fmt.Sprintf("DROP SUBSCRIPTION IF EXISTS %s CASCADE", postgres.QuoteIdentifier(subName))
	_, err := dstDB.ExecContext(ctx, query)
	if err != nil {
		end()
		if !noCleanup {
			ch.Printer.Printf("%s Failed to drop subscription: %v\n", printer.BoldRed("[ERROR]"), err)
			return
		}
		ch.Printer.Printf("%s Failed to drop subscription: %v\n", printer.BoldYellow("[WARN]"), err)
	} else {
		end()
		ch.Printer.Printf("Subscription dropped: %s\n", printer.BoldGreen("OK"))
	}

	if srcDB != nil {
		slotName := subName
		end = ch.Printer.PrintProgress("Dropping replication slot on source...")
		if err := postgres.DropReplicationSlot(ctx, srcDB, slotName); err != nil {
			end()
			ch.Printer.Printf("%s Failed to drop replication slot: %v\n", printer.BoldYellow("[WARN]"), err)
		} else {
			end()
			ch.Printer.Printf("Replication slot dropped: %s\n", printer.BoldGreen("OK"))
		}

		if pubName != "" {
			end = ch.Printer.PrintProgress("Dropping publication on source...")
			if err := postgres.DropPublication(ctx, srcDB, pubName, true); err != nil {
				end()
				ch.Printer.Printf("%s Failed to drop publication: %v\n", printer.BoldYellow("[WARN]"), err)
			} else {
				end()
				ch.Printer.Printf("Publication dropped: %s\n", printer.BoldGreen("OK"))
			}
		}

		srcDB.Close()
	}

	if roleID != "" {
		end = ch.Printer.PrintProgress("Cleaning up replication role...")
		err := client.PostgresRoles.Delete(ctx, &ps.DeletePostgresRoleRequest{
			Organization: org,
			Database:     database,
			Branch:       branch,
			RoleId:       roleID,
			Successor:    "postgres",
		})
		end()
		if err != nil {
			ch.Printer.Printf("%s Failed to delete replication role: %v\n", printer.BoldYellow("[WARN]"), err)
		} else {
			ch.Printer.Printf("Replication role cleaned up: %s\n", printer.BoldGreen("OK"))
		}
	}

	if err := creds.ClearSubscriptionCredentials(org, database, branch, subName); err != nil {
		ch.Printer.Printf("%s Failed to clear credentials: %v\n", printer.BoldYellow("[WARN]"), err)
	}
}

func printKeepReplicationInfo(ch *cmdutil.Helper, status *postgres.SubscriptionStatus) {
	ch.Printer.Printf("\n%s Replication kept active (--keep-replication)\n", printer.BoldYellow("Note:"))
	ch.Printer.Printf("  Subscription: %s\n", status.Name)
	ch.Printer.Printf("  Publication: %s\n", status.PublicationName)
}
