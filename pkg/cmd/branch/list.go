package branch

import (
	"context"
	"errors"
	"fmt"

	"github.com/pkg/browser"
	errs "github.com/pkg/errors"
	"github.com/planetscale/cli/cmdutil"
	"github.com/planetscale/cli/config"
	"github.com/planetscale/cli/printer"
	"github.com/spf13/cobra"
)

// ListCmd encapsulates the command for listing branches for a database.
func ListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <db-name>",
		Short: "List all branches of a database",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if len(args) == 0 {
				return errors.New("<db_name> is missing")
			}

			source := args[0]

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to your database branches in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s/branches", cmdutil.ApplicationURL, cfg.Organization, source))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			branches, err := client.DatabaseBranches.List(ctx, cfg.Organization, source)
			if err != nil {
				return errs.Wrap(err, "error listing databases")
			}

			// Technically, this should never actually happen.
			if len(branches) == 0 {
				fmt.Println("No branches exist for this database.")
				return nil
			}

			isJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}

			err = printer.PrintOutput(isJSON, printer.NewDatabaseBranchSlicePrinter(branches))
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolP("web", "w", false, "List database branches in your web browser.")
	return cmd
}
