package database

import (
	"context"
	"fmt"
	"net/url"

	"github.com/pkg/browser"
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
			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to create a database in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("https://app.planetscaledb.io/databases?slug=%s&label=%s&description=%s&showDialog=true", url.QueryEscape(createReq.Database.Slug), url.QueryEscape(createReq.Database.Label), url.QueryEscape(createReq.Database.Description)))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			database, err := client.Databases.Create(ctx, createReq)
			if err != nil {
				return err
			}

			fmt.Printf("Successfully created database: %s\n", database.Label)

			return nil
		},
	}

	cmd.Flags().StringVarP(&createReq.Database.Label, "label", "l", "", "the label for the database (required)")
	cmd.Flags().StringVarP(&createReq.Database.Slug, "slug", "s", "", "the slug for the database (required)")
	cmd.Flags().StringVarP(&createReq.Database.Description, "description", "d", "", "a description for the database (required)")
	_ = cmd.MarkFlagRequired("label")
	_ = cmd.MarkFlagRequired("slug")
	cmd.Flags().BoolP("web", "w", false, "Create a database in your web browser")

	return cmd
}
