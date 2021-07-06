package backup

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// ListCmd encapsulates the command for listing backups for a branch.
func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List all backups of a branch",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to your backups in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s/%s/backups", cmdutil.ApplicationURL, ch.Config.Organization, database, branch))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching backups for %s", printer.BoldBlue(branch)))
			defer end()
			backups, err := client.Backups.List(ctx, &planetscale.ListBackupsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(backups) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No backups exist in %s.\n", printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(toBackups(backups))
		},
	}

	cmd.Flags().BoolP("web", "w", false, "List backups in your web browser.")
	return cmd
}
