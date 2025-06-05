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
			
			// Validate that either password-id is provided as arg or --name flag is used
			var passwordId string
			if name != "" && len(args) == 3 {
				return errors.New("cannot specify both password-id argument and --name flag")
			}
			if name == "" && len(args) != 3 {
				return errors.New("must provide either password-id argument or --name flag")
			}
			
			client, err := ch.Client()
			if err != nil {
				return err
			}
			
			// If using --name flag, find the password by name
			if name != "" {
				end := ch.Printer.PrintProgress(fmt.Sprintf("Finding password %s in %s/%s",
					printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
				
				// Fetch all passwords to find the one with matching name
				var allPasswords []*ps.DatabaseBranchPassword
				page := 1
				perPage := 100
				
				for {
					passwords, err := client.Passwords.List(ctx, &ps.ListDatabaseBranchPasswordRequest{
						Organization: ch.Config.Organization,
						Database:     database,
						Branch:       branch,
					}, ps.WithPage(page), ps.WithPerPage(perPage))
					if err != nil {
						end()
						switch cmdutil.ErrCode(err) {
						case ps.ErrNotFound:
							return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
								printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
						default:
							return cmdutil.HandleError(err)
						}
					}
					
					allPasswords = append(allPasswords, passwords...)
					
					// Check if there are more pages
					if len(passwords) < perPage {
						break
					}
					page++
				}
				
				end()
				
				// Find password with matching name
				var foundPassword *ps.DatabaseBranchPassword
				for _, password := range allPasswords {
					if password.Name == name {
						foundPassword = password
						break
					}
				}
				
				if foundPassword == nil {
					return fmt.Errorf("password with name %s does not exist in branch %s of %s (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				}
				
				passwordId = foundPassword.PublicID
			} else {
				passwordId = args[2]
			}

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf("cannot delete password with the output format %q (run with -force to override)", ch.Printer.Format())
				}

				var confirmationName string
				if name != "" {
					confirmationName = fmt.Sprintf("%s/%s/%s", database, branch, name)
				} else {
					confirmationName = fmt.Sprintf("%s/%s/%s", database, branch, passwordId)
				}
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
