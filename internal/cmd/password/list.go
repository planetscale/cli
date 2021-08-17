package password

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// ListCmd encapsulates the command for listing passwords for a branch.
func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database> [branch]",
		Short:   "List all passwords of a database",
		Args:    cmdutil.RequiredArgs("database"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to your passwords in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s/settings/passwords", cmdutil.ApplicationURL, ch.Config.Organization, database))
				if err != nil {
					return err
				}
				return nil
			}

			var branch string
			if len(args) == 2 {
				branch = args[1]
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching passwords for %s", printer.BoldBlue(branch)))
			defer end()
			passwords, err := client.Passwords.List(ctx, &planetscale.ListDatabaseBranchPasswordRequest{
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

			if len(passwords) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No passwords exist in %s.\n", printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(toPasswords(passwords))
		},
	}

	cmd.Flags().BoolP("web", "w", false, "List passwords in your web browser.")
	return cmd
}
