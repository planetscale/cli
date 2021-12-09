package database

import (
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

// DeleteCmd is the Cobra command for deleting a database for an authenticated
// user.
func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database>",
		Short:   "Delete a database instance",
		Args:    cmdutil.RequiredArgs("database"),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf("cannot delete database with the output format %q (run with -force to override)", ch.Printer.Format())
				}

				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm deletion of database %q (run with -force to override)", name)
				}

				_, err := client.Databases.Get(ctx, &planetscale.GetDatabaseRequest{
					Organization: ch.Config.Organization,
					Database:     name,
				})
				if err != nil {
					switch cmdutil.ErrCode(err) {
					case planetscale.ErrNotFound:
						return fmt.Errorf("database %s does not exist in organization %s",
							printer.BoldBlue(name), printer.BoldBlue(ch.Config.Organization))
					default:
						return cmdutil.HandleError(err)
					}
				}

				confirmationMessage := fmt.Sprintf("%s %s %s", printer.Bold("Please type"), printer.BoldBlue(name), printer.Bold("to confirm:"))

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

				if userInput != name {
					return errors.New("incorrect database name entered, skipping database deletion")
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting database %s...", printer.BoldBlue(name)))
			defer end()

			err = client.Databases.Delete(ctx, &planetscale.DeleteDatabaseRequest{
				Organization: ch.Config.Organization,
				Database:     name,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(name), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Database %s was successfully deleted.\n", printer.BoldBlue(name))
				return nil
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result":   "database deleted",
					"database": name,
				},
			)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete a databse without confirmation")
	return cmd
}
