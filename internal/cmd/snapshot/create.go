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

func CreateCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a new schema snapshot for a database branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) != 2 {
				return cmd.Usage()
			}

			database, branch := args[0], args[1]

			end := cmdutil.PrintProgress(fmt.Sprintf("Creating schema snapshot for %s in %s...", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(database)))
			defer end()

			snapshot, err := client.SchemaSnapshots.Create(ctx, &planetscale.CreateSchemaSnapshotRequest{
				Organization: cfg.Organization,
				Database:     database,
				Branch:       branch,
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
