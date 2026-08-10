package backup

import (
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// PolicyDeleteCmd deletes a backup policy.
func PolicyDeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <policy-id>",
		Short:   "Delete a backup policy",
		Long:    "Delete a custom backup policy. Required default system policies cannot be deleted.",
		Args:    cmdutil.RequiredArgs("database", "policy-id"),
		Aliases: []string{"rm"},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			policyID := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			policy, err := client.BackupPolicies.Get(ctx, &ps.GetBackupPolicyRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Policy:       policyID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("backup policy %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(policyID), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if policy.Required {
				return fmt.Errorf("backup policy %s is a required default policy and cannot be deleted",
					printer.BoldBlue(policyID))
			}

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf(`cannot delete backup policy with the output format "%s" (run with --force to override)`, ch.Printer.Format())
				}

				confirmationName := fmt.Sprintf("%s/%s", database, policyID)
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm deletion of backup policy %q (run with --force to override)", confirmationName)
				}

				confirmationMessage := fmt.Sprintf("%s %s %s", printer.Bold("Please type"), printer.BoldBlue(confirmationName), printer.Bold("to confirm:"))
				prompt := &survey.Input{Message: confirmationMessage}

				var userInput string
				err := survey.AskOne(prompt, &userInput)
				if err != nil {
					if err == terminal.InterruptErr {
						os.Exit(0)
					}
					return err
				}

				if userInput != confirmationName {
					return errors.New("incorrect backup policy name entered, skipping deletion")
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting backup policy %s from %s", printer.BoldBlue(policyID), printer.BoldBlue(database)))
			defer end()

			err = client.BackupPolicies.Delete(ctx, &ps.DeleteBackupPolicyRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Policy:       policyID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("backup policy %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(policyID), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Backup policy %s was successfully deleted from %s.\n",
					printer.BoldBlue(policyID), printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result": "backup policy deleted",
				"policy": policyID,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete a backup policy without confirmation")
	return cmd
}
