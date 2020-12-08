package database

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/config"
	"github.com/planetscale/cli/psapi"
	"github.com/spf13/cobra"
)

// CreateCmd is the command for creating a database.
func CreateCmd(cfg *config.Config) *cobra.Command {
	createReq := &psapi.CreateDatabaseRequest{
		Database: new(psapi.Database),
	}
	cmd := &cobra.Command{
		Use: "create",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			database, err := client.Databases.Create(ctx, createReq)
			if err != nil {
				return err
			}

			fmt.Printf("Successfully created database: %s\n", database.Name)

			return nil
		},
	}

	cmd.Flags().StringVarP(&createReq.Database.Name, "name", "n", "", "the name of the database (required)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
