package snapshot

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a new schema snapshot for a database branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database, branch := args[0], args[1]

			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating schema snapshot for %s in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			snapshot, err := client.SchemaSnapshots.Create(ctx, &planetscale.CreateSchemaSnapshotRequest{
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
					return err
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Schema snapshot %s was successfully created!\n",
					printer.BoldBlue(snapshot.Name))
				return nil
			}

			return ch.Printer.PrintResource(toSchemaSnapshot(snapshot))
		},
	}

	return cmd
}
