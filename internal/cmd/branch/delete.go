package branch

import (
	"context"
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
			ctx := context.Background()
			source := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf("Cannot delete branch with the output format %q (run with -force to override)", ch.Printer.Format())
				}

				confirmationName := fmt.Sprintf("%s/%s", source, branch)
				if !printer.IsTTY {
					return fmt.Errorf("Cannot confirm deletion of branch %q (run with -force to override)", confirmationName)
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
					return errors.New("Incorrect database and branch name entered, skipping branch deletion...")
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting branch %s from %s", printer.BoldBlue(branch), printer.BoldBlue(source)))
			defer end()
			err = client.DatabaseBranches.Delete(ctx, &planetscale.DeleteDatabaseBranchRequest{
				Organization: ch.Config.Organization,
				Database:     source,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("source database %s does not exist in organization %s\n",
						printer.BoldBlue(source), printer.BoldBlue(ch.Config.Organization))
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
