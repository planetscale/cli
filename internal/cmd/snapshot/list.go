package snapshot

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ListCmd makes a command for listing all snapshots for a database branch.
func ListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List all of the schema snapshots for a database branch",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database, branch := args[0], args[1]

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Fetching schema snapshots for %s in %s...", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(database)))
			defer end()

			snapshots, err := client.SchemaSnapshots.List(ctx, &planetscale.ListSchemaSnapshotsRequest{
				Organization: cfg.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						cmdutil.BoldBlue(branch), cmdutil.BoldBlue(database), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return errors.Wrap(err, "error listing schema snapshots")
				}
			}
			end()

			if len(snapshots) == 0 && !cfg.OutputJSON {
				fmt.Printf("No schema snapshots exist for %s in %s.\n", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(database))
				return nil
			}

			err = printer.PrintOutput(cfg.OutputJSON, printer.NewSchemaSnapshotSlicePrinter(snapshots))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
