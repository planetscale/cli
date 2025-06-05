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

			client, err := ch.Client()
			if err != nil {
				return err
			}

			forMsg := printer.BoldBlue(database)

			var branch string
			if len(args) == 2 {
				branch = args[1]
				forMsg = fmt.Sprintf("%s/%s", printer.BoldBlue(database), printer.BoldBlue(branch))
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching passwords for %s", forMsg))
			defer end()

			var allPasswords []*planetscale.DatabaseBranchPassword
			page := 1
			perPage := 100
			
			for {
				passwords, err := client.Passwords.List(ctx, &planetscale.ListDatabaseBranchPasswordRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Branch:       branch,
				}, planetscale.WithPage(page), planetscale.WithPerPage(perPage))
				if err != nil {
					switch cmdutil.ErrCode(err) {
					case planetscale.ErrNotFound:
						return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
							printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
					default:
						return cmdutil.HandleError(err)
					}
				}
				
				allPasswords = append(allPasswords, passwords...)
				
				// Check if there are more pages - if we got fewer results than perPage, we're done
				if len(passwords) < perPage {
					break
				}
				page++
			}
			
			passwords := allPasswords
			end()

			if len(passwords) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No passwords exist in %s.\n", forMsg)
				return nil
			}

			// if we're doing human display and none of our passwords are ephemeral
			// we can hide a few of the columns for a more compact view.
			if ch.Printer.Format() == printer.Human && !hasEphemeral(passwords) {
				return ch.Printer.PrintResource(toPasswordsWithoutTTL(passwords))
			}

			return ch.Printer.PrintResource(toPasswords(passwords))
		},
	}

	cmd.Flags().BoolP("web", "w", false, "List passwords in your web browser.")
	return cmd
}
