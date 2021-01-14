package database

import (
	"context"
	"errors"
	"os"

	"github.com/lensesio/tableprinter"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// StatusCmd encapsulates the
func StatusCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <database_name>",
		Short: "Get the status of a database",
		Args:  cobra.ExactArgs(1),
		Long:  "TODO",
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

			status, err := client.Databases.Status(ctx, cfg.Organization, name)
			if err != nil {
				return err
			}

			tableprinter.Print(os.Stdout, status)
			return nil
		},
	}

	return cmd
}
