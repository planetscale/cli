package branch

import (
	"context"
	"os"

	"github.com/lensesio/tableprinter"
	"github.com/planetscale/cli/config"
	"github.com/planetscale/cli/printer"
	"github.com/spf13/cobra"
)

func GetCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <source_name> <branch_name>",
		Short: "Get a specific branch of a database",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if len(args) != 2 {
				return cmd.Usage()
			}

			source := args[0]
			branch := args[1]

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			b, err := client.DatabaseBranches.Get(ctx, cfg.Organization, source, branch)
			if err != nil {
				return err
			}

			tableprinter.Print(os.Stdout, printer.NewDatabaseBranchPrinter(b))

			return nil
		},
	}

	return cmd
}
