package password

import (
	"errors"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	ps "github.com/planetscale/cli/internal/planetscale"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
)

func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool
	var name string

	cmd := &cobra.Command{
		Use:     "delete <database> <branch> [<password-id>]",
		Short:   "Delete a branch password",
		Args:    cobra.RangeArgs(2, 3),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			if err := passwordSelector(args, name); err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			var argID string
			if len(args) == 3 {
				argID = args[2]
			}

			passwordId, err := resolvePasswordID(ctx, ch, client, database, branch, argID, name)
			if err != nil {
				return err
			}

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf(`cannot delete password with the output format "%s" (run with --force to override)`, ch.Printer.Format())
				}

				var confirmationName string
				if name != "" {
					confirmationName = fmt.Sprintf("%s/%s/%s", database, branch, name)
				} else {
					confirmationName = fmt.Sprintf("%s/%s/%s", database, branch, passwordId)
				}
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm deletion of password %q (run with --force to override)", confirmationName)
				}

				confirmationMessage := fmt.Sprintf("%s %s %s", printer.Bold("Please type"), printer.BoldBlue(confirmationName), printer.Bold("to confirm:"))

				prompt := &survey.Input{
					Message: confirmationMessage,
				}

				var userInput string
				err := survey.AskOne(prompt, &userInput)
				if err != nil {
					if err == terminal.InterruptErr {
						os.Exit(0)
					} else {
						return err
					}
				}

				// If the confirmations don't match up, let's return an error.
				if userInput != confirmationName {
					return errors.New("incorrect password name entered, skipping password deletion")
				}
			}

			var deleteMsg string
			if name != "" {
				deleteMsg = fmt.Sprintf("Deleting password %s from %s/%s",
					printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch))
			} else {
				deleteMsg = fmt.Sprintf("Deleting password %s from %s/%s",
					printer.BoldBlue(passwordId), printer.BoldBlue(database), printer.BoldBlue(branch))
			}
			end := ch.Printer.PrintProgress(deleteMsg)
			defer end()

			err = client.Passwords.Delete(ctx, &ps.DeleteDatabaseBranchPasswordRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				PasswordId:   passwordId,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("password %s does not exist in branch %s of %s (organization: %s)",
						printer.BoldBlue(passwordId), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				if name != "" {
					ch.Printer.Printf("Password %s was successfully deleted from %s.\n",
						printer.BoldBlue(name), printer.BoldBlue(branch))
				} else {
					ch.Printer.Printf("Password %s was successfully deleted from %s.\n",
						printer.BoldBlue(passwordId), printer.BoldBlue(branch))
				}
				return nil
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result":      "password deleted",
					"password_id": passwordId,
					"branch":      branch,
				},
			)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete a password without confirmation")
	cmd.Flags().StringVar(&name, "name", "", "Delete password by name instead of ID")
	return cmd
}
