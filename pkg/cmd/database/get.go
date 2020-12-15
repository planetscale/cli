package database

import (
	"context"
	"os"
	"strconv"

	"github.com/lensesio/tableprinter"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

func GetCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <databases_id>",
		Short: "Retrieve information about a database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			id, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}

			database, err := client.Databases.Get(ctx, int64(id))
			if err != nil {
				return err
			}

			tableprinter.Print(os.Stdout, database)
			return nil
		},
	}

	return cmd
}
