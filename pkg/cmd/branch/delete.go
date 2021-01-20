package branch

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/fatih/color"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

func DeleteCmd(cfg *config.Config) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <db_name> <branch_name>",
		Short:   "Delete a branch from a database",
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if len(args) != 2 {
				return cmd.Usage()
			}

			source := args[0]
			branch := args[1]

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			boldBlue := color.New(color.FgBlue).Add(color.Bold).SprintFunc()
			bold := color.New(color.Bold).SprintFunc()

			if !force {
				confirmationName := fmt.Sprintf("%s/%s", source, branch)
				userInput := ""

				confirmationMessage := fmt.Sprintf("%s %s %s", bold("Please type"), boldBlue(confirmationName), bold("to confirm:"))

				prompt := &survey.Input{
					Message: confirmationMessage,
				}

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

			err = client.DatabaseBranches.Delete(ctx, cfg.Organization, source, branch)
			if err != nil {
				return err
			}

			fmt.Printf("Successfully deleted branch %s from %s\n", boldBlue(branch), boldBlue(source))

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Delete a databse without confirmation")
	return cmd
}
