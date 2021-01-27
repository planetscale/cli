package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/pkg/browser"
	"github.com/planetscale/cli/cmdutil"
	"github.com/planetscale/cli/config"
	"github.com/planetscale/cli/printer"
	ps "github.com/planetscale/planetscale-go"

	"github.com/spf13/cobra"
)

// CreateCmd is the command for creating a database.
func CreateCmd(cfg *config.Config) *cobra.Command {
	createReq := &ps.CreateDatabaseRequest{
		Database: new(ps.Database),
	}

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a database instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if len(args) != 1 {
				return errors.New("<name> is missing")
			}

			createReq.Database.Name = args[0]
			createReq.Organization = cfg.Organization

			if web {
				fmt.Println("🌐  Redirecting you to create a database in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s?name=%s&notes=%s&showDialog=true", cmdutil.ApplicationURL, cfg.Organization, url.QueryEscape(createReq.Database.Name), url.QueryEscape(createReq.Database.Notes)))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress("Creating database...")
			defer end()
			database, err := client.Databases.Create(ctx, createReq)
			if err != nil {
				return err
			}

			isJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}

			end()
			if isJSON {
				err := printer.PrintJSON(database)
				if err != nil {
					return err
				}
			} else {
				fmt.Printf("Database %s was successfully created!\n", cmdutil.BoldBlue(database.Name))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&createReq.Database.Notes, "notes", "", "notes for the database")
	cmd.Flags().BoolP("web", "w", false, "Create a database in your web browser")

	return cmd
}
