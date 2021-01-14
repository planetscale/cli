package branch

import (
	"context"
	"os"

	"github.com/lensesio/tableprinter"
	"github.com/planetscale/cli/config"
	"github.com/planetscale/cli/printer"
	"github.com/spf13/cobra"
)

// StatusCmd gets the status of a database branch using the PlanetScale API.
func StatusCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <db_name> <branch_name>",
		Short: "Check the status of a branch of a database",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if len(args) != 2 {
				return cmd.Usage()
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			source := args[0]
			branch := args[1]

			// TODO: Call PlanetScale API here to get database branch status.
			status, err := client.DatabaseBranches.Status(ctx, cfg.Organization, source, branch)
			if err != nil {
				return err
			}

			tableprinter.Print(os.Stdout, printer.NewDatabaseBranchStatusPrinter(status))

			return nil
		},
	}

	return cmd
}
