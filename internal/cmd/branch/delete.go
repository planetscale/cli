package branch

import (
	"errors"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
)

func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <branch>",
		Short:   "Delete a branch from a database",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			source := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			db, err := client.Databases.Get(ctx, &planetscale.GetDatabaseRequest{
				Organization: ch.Config.Organization,
				Database:     source,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err, "delete_branch",
						"database %s does not exist in organization %s",
						printer.BoldBlue(source), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf("cannot delete branch with the output format %q (run with -force to override)", ch.Printer.Format())
				}

				if db.Kind == "mysql" {
					_, err := client.DatabaseBranches.Get(ctx, &planetscale.GetDatabaseBranchRequest{
						Organization: ch.Config.Organization,
						Database:     source,
						Branch:       branch,
					})
					if err != nil {
						switch cmdutil.ErrCode(err) {
						case planetscale.ErrNotFound:
							return cmdutil.HandleNotFoundWithServiceTokenCheck(
								ctx, cmd, ch.Config, ch.Client, err, "delete_branch",
								"branch %s does not exist in database %s (organization: %s)",
								printer.BoldBlue(branch), printer.BoldBlue(source), printer.BoldBlue(ch.Config.Organization))
						default:
							return cmdutil.HandleError(err)
						}
					}
				} else {
					_, err := client.PostgresBranches.Get(ctx, &planetscale.GetPostgresBranchRequest{
						Organization: ch.Config.Organization,
						Database:     source,
						Branch:       branch,
					})
					if err != nil {
						switch cmdutil.ErrCode(err) {
						case planetscale.ErrNotFound:
							return cmdutil.HandleNotFoundWithServiceTokenCheck(
								ctx, cmd, ch.Config, ch.Client, err, "delete_branch",
								"branch %s does not exist in database %s (organization: %s)",
								printer.BoldBlue(branch), printer.BoldBlue(source), printer.BoldBlue(ch.Config.Organization))
						default:
							return cmdutil.HandleError(err)
						}
					}
				}

				confirmationName := fmt.Sprintf("%s/%s", source, branch)
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm deletion of branch %q (run with -force to override)", confirmationName)
				}

				confirmationMessage := fmt.Sprintf("%s %s %s", printer.Bold("Please type"),
					printer.BoldBlue(confirmationName), printer.Bold("to confirm:"))

				prompt := &survey.Input{
					Message: confirmationMessage,
				}

				var userInput string
				err = survey.AskOne(prompt, &userInput)
				if err != nil {
					if err == terminal.InterruptErr {
						os.Exit(0)
					} else {
						return err
					}
				}

				// If the confirmations don't match up, let's return an error.
				if userInput != confirmationName {
					return errors.New("incorrect database and branch name entered, skipping branch deletion")
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting branch %s from %s", printer.BoldBlue(branch), printer.BoldBlue(source)))
			defer end()

			if db.Kind == "mysql" {
				err = client.DatabaseBranches.Delete(ctx, &planetscale.DeleteDatabaseBranchRequest{
					Organization: ch.Config.Organization,
					Database:     source,
					Branch:       branch,
				})
			} else {
				err = client.PostgresBranches.Delete(ctx, &planetscale.DeletePostgresBranchRequest{
					Organization: ch.Config.Organization,
					Database:     source,
					Branch:       branch,
				})
			}

			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err, "delete_branch",
						"database %s or branch %s does not exist in organization %s",
						printer.BoldBlue(source), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Branch %s was successfully deleted from %s.\n", printer.BoldBlue(branch), printer.BoldBlue(source))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{"result": "branch deleted", "branch": branch})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete a branch without confirmation")
	return cmd
}
