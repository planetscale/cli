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

func ReassignCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		force     bool
		successor string
	}

	cmd := &cobra.Command{
		Use:   "reassign <database> <branch> <role-id>",
		Short: "Reassign objects owned by a role to another role",
		Args:  cmdutil.RequiredArgs("database", "branch", "role-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			roleID := args[2]

			if flags.successor == "" {
				return fmt.Errorf("--successor flag is required")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !flags.force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf("cannot reassign role objects with the output format \"%s\" (run with --force to override)", ch.Printer.Format())
				}

				confirmationName := fmt.Sprintf("%s/%s/%s", database, branch, roleID)
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm object reassignment for role %q (run with --force to override)", confirmationName)
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
					return errors.New("incorrect role identifier entered, skipping object reassignment")
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Reassigning objects from role %s to %s in %s/%s...",
				printer.BoldBlue(roleID), printer.BoldBlue(flags.successor), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			err = client.PostgresRoles.ReassignObjects(ctx, &ps.ReassignPostgresRoleObjectsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				RoleId:       roleID,
				Successor:    flags.successor,
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
				ch.Printer.Printf("Objects owned by role %s were successfully reassigned to %s in %s/%s.\n",
					printer.BoldBlue(roleID), printer.BoldBlue(flags.successor), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result":    "objects reassigned",
					"role_id":   roleID,
					"successor": flags.successor,
					"branch":    branch,
				},
			)
		},
	}

	cmd.Flags().BoolVar(&flags.force, "force", false, "Reassign objects without confirmation")
	cmd.Flags().StringVar(&flags.successor, "successor", "", "Role to transfer ownership to (required)")
	cmd.MarkFlagRequired("successor") // nolint:errcheck

	return cmd
}
