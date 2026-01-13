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

func ResetDefaultCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		force bool
	}

	cmd := &cobra.Command{
		Use:   "reset-default <database> <branch>",
		Short: "Reset the credentials for the default `postgres` role",
		Long:  "This command resets the credentials for the default `postgres` role in the database, allowing you to reconfigure access. Any connections using the `postgres` role will need to be updated with the new credentials.",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			org := ch.Config.Organization
			database := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !flags.force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf("cannot delete password with the output format %q (run with --force to override)", ch.Printer.Format())
				}

				confirmationName := fmt.Sprintf("%s/%s", database, branch)
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm deletion of branch %q (run with --force to override)", confirmationName)
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
					return errors.New("incorrect database and branch name entered, skipping reset")
				}

			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Resetting default postgres role for %s/%s...", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			role, err := client.PostgresRoles.ResetDefaultRole(cmd.Context(), &ps.ResetDefaultRoleRequest{
				Organization: org,
				Database:     database,
				Branch:       branch,
			})
		if err != nil {
			switch cmdutil.ErrCode(err) {
			case ps.ErrNotFound:
				return cmdutil.HandleNotFoundWithServiceTokenCheck(
					cmd.Context(), cmd, ch.Config, ch.Client, err,
					"delete_branch_password or delete_production_branch_password",
					"database %s or branch %s does not exist in organization %s",
					printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
			default:
				return cmdutil.HandleError(err)
			}
		}

			end()

			if ch.Printer.Format() == printer.Human {
				saveWarning := printer.BoldRed("Please save the values below as they will not be shown again. We recommend using these credentials only for creating usernames and passwords for accessing your database.")

				ch.Printer.Printf("Role was successfully reset for %s in %s.\n%s\n\n", printer.BoldBlue(branch), printer.BoldBlue(database), saveWarning)
			}

			return ch.Printer.PrintResource(toPostgresRole(role))
		},
	}

	cmd.Flags().BoolVar(&flags.force, "force", false, "Force reset without confirmation")

	return cmd
}

type PostgresRole struct {
	PublicID      string `header:"id" json:"id"`
	Name          string `header:"name" json:"name"`
	Username      string `header:"username" json:"username"`
	Password      string `header:"password" json:"password"`
	AccessHostURL string `header:"access_host_url" json:"access_host_url"`

	orig *ps.PostgresRole
}

func toPostgresRole(role *ps.PostgresRole) *PostgresRole {
	return &PostgresRole{
		PublicID:      role.ID,
		Name:          role.Name,
		Username:      role.Username,
		Password:      role.Password,
		AccessHostURL: role.AccessHostURL,
		orig:          role,
	}
}
