package database

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

// DeleteCmd is the Cobra command for deleting a database for an authenticated
// user.
func DeleteCmd(cfg *config.Config) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database_name>",
		Short:   "Delete a database instance",
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return errors.New("<database_name> is missing")
			}

			name := args[0]

			boldBlue := color.New(color.FgBlue).Add(color.Bold).SprintFunc()
			bold := color.New(color.Bold).SprintfFunc()

			if !force {
				userInput := ""
				confirmationMessage := fmt.Sprintf("%s %s %s", bold("Please type"), boldBlue(name), bold("to confirm:"))

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

				if userInput != name {
					return errors.New("Incorrect database name entered, skipping database deletion...")
				}
			}

			deleted, err := client.Databases.Delete(ctx, cfg.Organization, name)
			if err != nil {
				return err
			}

			if deleted {
				fmt.Printf("Successfully deleted database %s\n", boldBlue(name))
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Delete a databse without confirmation")
	return cmd
}
