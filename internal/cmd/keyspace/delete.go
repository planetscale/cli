package keyspace

import (
	"errors"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
)

// DeleteCmd encapsulates the command for deleting a keyspace from a branch.
func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <branch> <keyspace>",
		Short:   "Delete a keyspace from a branch",
		Args:    cmdutil.RequiredArgs("database", "branch", "keyspace"),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf(`cannot delete keyspace with the output format "%s" (run with --force to override)`, ch.Printer.Format())
				}

				_, err := client.Keyspaces.Get(ctx, &planetscale.GetKeyspaceRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Branch:       branch,
					Keyspace:     keyspace,
				})
				if err != nil {
					switch cmdutil.ErrCode(err) {
					case planetscale.ErrNotFound:
						return fmt.Errorf("keyspace %s does not exist in branch %s of %s (organization: %s)",
							printer.BoldBlue(keyspace), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
					default:
						return cmdutil.HandleError(err)
					}
				}

				confirmationName := fmt.Sprintf("%s/%s/%s", database, branch, keyspace)
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm deletion of keyspace %q (run with --force to override)", confirmationName)
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
					}
					return err
				}

				if userInput != confirmationName {
					return errors.New("incorrect database, branch, and keyspace name entered, skipping keyspace deletion")
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting keyspace %s from %s/%s",
				printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			err = client.Keyspaces.Delete(ctx, &planetscale.DeleteKeyspaceRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     keyspace,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("keyspace %s does not exist in branch %s of %s (organization: %s)",
						printer.BoldBlue(keyspace), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Keyspace %s was successfully deleted from %s/%s.\n",
					printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result":   "keyspace deleted",
				"keyspace": keyspace,
				"branch":   branch,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete a keyspace without confirmation")
	return cmd
}
