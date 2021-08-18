package password

import (
	"errors"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
)

func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <branch> <password>",
		Short:   "Delete a branch password",
		Args:    cmdutil.RequiredArgs("database", "branch", "password"),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			password := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf("cannot delete password with the output format %q (run with -force to override)", ch.Printer.Format())
				}

				confirmationName := fmt.Sprintf("%s/%s/%s", database, branch, password)
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm deletion of password %q (run with -force to override)", confirmationName)
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting password %s from %s/%s",
				printer.BoldBlue(password), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			err = client.Passwords.Delete(ctx, &ps.DeleteDatabaseBranchPasswordRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				PasswordId:   password,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("password %s does not exist in branch %s of %s (organization: %s)",
						printer.BoldBlue(password), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Password %s was successfully deleted from %s.\n",
					printer.BoldBlue(password), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result":   "password deleted",
					"password": password,
					"branch":   branch,
				},
			)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete a password without confirmation")
	return cmd
}
