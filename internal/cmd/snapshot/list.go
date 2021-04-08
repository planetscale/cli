package snapshot

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ListCmd makes a command for listing all snapshots for a database branch.
func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List all of the schema snapshots for a database branch",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database, branch := args[0], args[1]

			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching schema snapshots for %s in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			snapshots, err := client.SchemaSnapshots.List(ctx, &planetscale.ListSchemaSnapshotsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return errors.Wrap(err, "error listing schema snapshots")
				}
			}
			end()

			if len(snapshots) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No schema snapshots exist for %s in %s.\n",
					printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(toSchemaSnapshots(snapshots))
		},
	}

	return cmd
}
