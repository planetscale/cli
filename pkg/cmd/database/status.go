package database

import (
	"context"
	"os"
	"strconv"

	"github.com/lensesio/tableprinter"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// StatusCmd encapsulates the
func StatusCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <database_id>",
		Short: "Get the status of a database",
		Args:  cobra.ExactArgs(1),
		Long:  "TODO",
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

			status, err := client.Databases.Status(ctx, int64(id))
			if err != nil {
				return err
			}

			tableprinter.Print(os.Stdout, status)
			return nil
		},
	}

	return cmd
}
