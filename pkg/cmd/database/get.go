package database

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/lensesio/tableprinter"
	"github.com/pkg/browser"
	"github.com/planetscale/cli/cmdutil"
	"github.com/planetscale/cli/config"
	"github.com/planetscale/cli/printer"
	"github.com/spf13/cobra"
)

func GetCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <database_name>",
		Short: "Retrieve information about a database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return errors.New("<database_name> is missing")
			}

			name := args[0]

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
			database, err := client.Databases.Get(ctx, cfg.Organization, name)
			if err != nil {
				return err
			}

			tableprinter.Print(os.Stdout, printer.NewDatabasePrinter(database))
			return nil
		},
	}

	cmd.Flags().BoolP("web", "w", false, "Open in your web browser")

	return cmd
}
