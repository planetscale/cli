package role

import (
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func ResetCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		force bool
	}

	cmd := &cobra.Command{
		Use:   "reset <database> <branch> <role-id>",
		Short: "Reset a role's password",
		Args:  cmdutil.RequiredArgs("database", "branch", "role-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			roleID := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !flags.force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf(`cannot reset role password with the output format "%s" (run with --force to override)`, ch.Printer.Format())
				}

				confirmationName := fmt.Sprintf("%s/%s/%s", database, branch, roleID)
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm password reset for role %q (run with --force to override)", confirmationName)
				}

				confirmationMessage := fmt.Sprintf("%s %s %s", printer.Bold("Please type"),
					printer.BoldBlue(confirmationName), printer.Bold("to confirm:"))

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
					return errors.New("incorrect role identifier entered, skipping password reset")
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Resetting password for role %s in %s/%s...",
				printer.BoldBlue(roleID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			role, err := client.PostgresRoles.ResetPassword(ctx, &ps.ResetPostgresRolePasswordRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				RoleId:       roleID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err,
						"delete_branch_password or delete_production_branch_password",
						"role %s does not exist in branch %s of database %s (organization: %s)",
						printer.BoldBlue(roleID), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Password for role %s was successfully reset in %s/%s.\n\n",
					printer.BoldBlue(roleID), printer.BoldBlue(database), printer.BoldBlue(branch))
				printPostgresRoleCredentials(ch.Printer, toPostgresRole(role))
				return nil
			}

			return ch.Printer.PrintResource(toPostgresRole(role))
		},
	}

	cmd.Flags().BoolVar(&flags.force, "force", false, "Reset password without confirmation")

	return cmd
}
