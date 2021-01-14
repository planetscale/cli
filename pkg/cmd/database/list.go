package database

import (
	"context"
	"fmt"
	"os"

	"github.com/lensesio/tableprinter"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// ListCmd is the command for listing all databases for an authenticated user.
func ListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List databases",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to your databases list in your web browser.")
				err := browser.OpenURL("https://app.planetscaledb.io/databases")
				if err != nil {
					return err
				}
				return nil
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			databases, err := client.Databases.List(ctx, cfg.Organization)
			if err != nil {
				return errors.Wrap(err, "error listing databases")
			}

			if len(databases) == 0 {
				fmt.Println("No databases have been created yet.")
				return nil
			}

			tableprinter.Print(os.Stdout, databases)
			return nil
		},
		TraverseChildren: true,
	}

	cmd.Flags().BoolP("web", "w", false, "Open in your web browser")

	return cmd
}
