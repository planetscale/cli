package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// DeleteCmd is the Cobra command for deleting a database for an authenticated
// user.
func DeleteCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "delete <database_id>",
		Args: cobra.ExactArgs(1),
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

			deleted, err := client.Databases.Delete(ctx, int64(id))
			if err != nil {
				return err
			}

			if deleted {
				fmt.Printf("Successfully deleted database with ID: %d\n", id)
			}

			return nil
		},
	}

	return cmd
}
