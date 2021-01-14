package database

import (
	"context"
	"errors"
	"os"

	"github.com/lensesio/tableprinter"
	"github.com/planetscale/cli/config"
	"github.com/planetscale/cli/printer"
	"github.com/spf13/cobra"
)

func GetCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <databases_name>",
		Short: "Retrieve information about a database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return errors.New("<database_name> is missing")
			}
			name := args[0]

			database, err := client.Databases.Get(ctx, cfg.Organization, name)
			if err != nil {
				return err
			}

			tableprinter.Print(os.Stdout, printer.NewDatabasePrinter(database))
			return nil
		},
	}

	return cmd
}
