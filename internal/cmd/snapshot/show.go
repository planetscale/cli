package snapshot

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ShowCmd makes a command for fetching a single snapshot by its ID.
func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <snapshot-id>",
		Short: "Show a specific schema snapshot",
		Args:  cmdutil.RequiredArgs("snapshot-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			id := args[0]

			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching schema snapshot %s", printer.BoldBlue(id)))
			defer end()

			snapshot, err := client.SchemaSnapshots.Get(ctx, &planetscale.GetSchemaSnapshotRequest{
				ID: id,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("snapshot id %s does not exist (organization: %s)", printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}
			end()

			return ch.Printer.PrintResource(toSchemaSnapshot(snapshot))
		},
	}

	return cmd
}
