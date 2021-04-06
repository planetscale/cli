package branch

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"

	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
)

func DeleteCmd(cfg *config.Config) *cobra.Command {
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

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if !force {
				confirmationName := fmt.Sprintf("%s/%s", source, branch)
				if !cmdutil.IsTTY {
					return fmt.Errorf("Cannot confirm deletion of branch %q (run with -force to override)", confirmationName)
				}

				confirmationMessage := fmt.Sprintf("%s %s %s", cmdutil.Bold("Please type"), cmdutil.BoldBlue(confirmationName), cmdutil.Bold("to confirm:"))

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

			end := cmdutil.PrintProgress(fmt.Sprintf("Deleting branch %s from %s", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(source)))
			defer end()
			err = client.DatabaseBranches.Delete(ctx, &planetscale.DeleteDatabaseBranchRequest{
				Organization: cfg.Organization,
				Database:     source,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("%s does not exist in %s\n", cmdutil.BoldBlue(source), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()
			fmt.Printf("Branch %s was successfully deleted from %s!\n", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(source))

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Delete a branch without confirmation")
	return cmd
}
