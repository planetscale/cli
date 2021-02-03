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
		Use:   "get <snapshot_id>",
		Short: "Get a specific schema snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) != 1 {
				return cmd.Usage()
			}

			id := args[0]
			end := cmdutil.PrintProgress(fmt.Sprintf("Fetching schema snapshot %s", cmdutil.BoldBlue(id)))
			defer end()

			snapshot, err := client.SchemaSnapshots.Get(ctx, &planetscale.GetSchemaSnapshotRequest{
				ID: id,
			})
			if err != nil {
				return err
			}
			end()

			isJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}

			err = printer.PrintOutput(isJSON, printer.NewSchemaSnapshotPrinter(snapshot))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
