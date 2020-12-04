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
	// TODO(notfelineit/iheanyi): Add `--web` flag for opening the list of
	// databases here in the web UI.
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			web, _ := cmd.Flags().GetBool("web")
			if web == true {
				fmt.Println("🌐  Redirecting you to your databases list in your web browser.")
				browser.OpenURL("https://planetscale-app-bb.vercel.app/databases")
				return nil
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			databases, err := client.Databases.List(ctx)
			if err != nil {
				return errors.Wrap(err, "error listing databases")
			}

			tableprinter.Print(os.Stdout, databases)
			return nil
		},
		TraverseChildren: true,
	}

	cmd.Flags().BoolP("web", "w", false, "Open in your web browser")

	return cmd
}
