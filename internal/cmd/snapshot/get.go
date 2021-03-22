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

// GetCmd makes a command for fetching a single snapshot by its ID.
func GetCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <snapshot-id>",
		Short: "Get a specific schema snapshot",
		Args:  cmdutil.RequiredArgs("snapshot-id"),
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
				return err
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
