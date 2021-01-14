package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// DeleteCmd is the Cobra command for deleting a database for an authenticated
// user.
func DeleteCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <database_name>",
		Short: "Delete a database instance",
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

			deleted, err := client.Databases.Delete(ctx, cfg.Organization, name)
			if err != nil {
				return err
			}

			if deleted {
				fmt.Printf("Successfully deleted database with the name: %q\n", name)
			}

			return nil
		},
	}

	return cmd
}
