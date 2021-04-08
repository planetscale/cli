package branch

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ListCmd encapsulates the command for listing branches for a database.
func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database>",
		Short:   "List all branches of a database",
		Args:    cmdutil.RequiredArgs("database"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database := args[0]

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				ch.Printer.Println("🌐  Redirecting you to your branches in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s/branches", cmdutil.ApplicationURL, ch.Config.Organization, database))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching branches for %s", printer.BoldBlue(database)))
			defer end()

			branches, err := client.DatabaseBranches.List(ctx, &planetscale.ListDatabaseBranchesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s\n",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return errors.Wrap(err, "error listing branches")
				}
			}
			end()

			if len(branches) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No branches exist in %s.\n", printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(toDatabaseBranches(branches))
		},
	}

	cmd.Flags().BoolP("web", "w", false, "List branches in your web browser.")
	return cmd
}
