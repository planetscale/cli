package database

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func GetCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <database>",
		Short: "Retrieve information about a database",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			name := args[0]

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to your database in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s", cmdutil.ApplicationURL, cfg.Organization, name))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Fetching database %s...", cmdutil.BoldBlue(name)))
			defer end()
			database, err := client.Databases.Get(ctx, &planetscale.GetDatabaseRequest{
				Organization: cfg.Organization,
				Database:     name,
			})
			if err != nil {
				return err
			}

			isJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}

			end()
			err = printer.PrintOutput(isJSON, printer.NewDatabasePrinter(database))
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolP("web", "w", false, "Open in your web browser")

	return cmd
}
