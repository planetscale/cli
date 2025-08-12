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

func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		force     bool
		successor string
	}

	cmd := &cobra.Command{
		Use:     "delete <database> <branch> <role-id>",
		Short:   "Delete a role",
		Args:    cmdutil.RequiredArgs("database", "branch", "role-id"),
		Aliases: []string{"rm"},
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
					return fmt.Errorf("cannot delete role with the output format %q (run with --force to override)", ch.Printer.Format())
				}

				confirmationName := fmt.Sprintf("%s/%s/%s", database, branch, roleID)
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm deletion of role %q (run with --force to override)", confirmationName)
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
					return errors.New("incorrect role identifier entered, skipping role deletion")
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting role %s from %s/%s...",
				printer.BoldBlue(roleID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			err = client.PostgresRoles.Delete(ctx, &ps.DeletePostgresRoleRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				RoleId:       roleID,
				Successor:    flags.successor,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("role %s does not exist in branch %s of database %s (organization: %s)",
						printer.BoldBlue(roleID), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Role %s was successfully deleted from %s/%s.\n",
					printer.BoldBlue(roleID), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result":  "role deleted",
					"role_id": roleID,
					"branch":  branch,
				},
			)
		},
	}

	cmd.Flags().BoolVar(&flags.force, "force", false, "Delete a role without confirmation")
	cmd.Flags().StringVar(&flags.successor, "successor", "", "Role to transfer ownership to before deletion. Usually 'postgres'.")

	return cmd
}
