package snapshot

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ShowCmd makes a command for fetching a single snapshot by its ID.
func ShowCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show <snapshot-id>",
		Short:   "Show a specific schema snapshot",
		Aliases: []string{"get"},
		Args:    cmdutil.RequiredArgs("snapshot-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			id := args[0]

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Fetching schema snapshot %s", cmdutil.BoldBlue(id)))
			defer end()

			snapshot, err := client.SchemaSnapshots.Get(ctx, &planetscale.GetSchemaSnapshotRequest{
				ID: id,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("snapshot id %s does not exist (organization: %s)", cmdutil.BoldBlue(id), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}
			end()

			err = printer.PrintOutput(cfg.OutputJSON, printer.NewSchemaSnapshotPrinter(snapshot))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
