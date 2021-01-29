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
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ListCmd encapsulates the command for listing branches for a database.
func ListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <db-name>",
		Short:   "List all branches of a database",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if len(args) == 0 {
				return errors.New("<db_name> is missing")
			}

			database := args[0]

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to your branches in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s/branches", cmdutil.ApplicationURL, cfg.Organization, database))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Fetching branches for %s", cmdutil.BoldBlue(database)))
			defer end()
			branches, err := client.DatabaseBranches.List(ctx, &planetscale.ListDatabaseBranchesRequest{
				Organization: cfg.Organization,
				Database:     database,
			})
			if err != nil {
				return errs.Wrap(err, "error listing branches")
			}
			end()

			isJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}

			if len(branches) == 0 && !isJSON {
				fmt.Printf("No branches exist in %s.\n", cmdutil.BoldBlue(database))
				return nil
			}

			err = printer.PrintOutput(isJSON, printer.NewDatabaseBranchSlicePrinter(branches))
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolP("web", "w", false, "List branches in your web browser.")
	return cmd
}
