package dataimports

import (
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
	"os"
)

func CancelDataImportCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name  string
		force bool
	}

	cancelRequest := &ps.CancelDataImportRequest{}

	cmd := &cobra.Command{
		Use:     "cancel [database]",
		Short:   "cancel data import request into a planetscale database",
		Aliases: []string{"c"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if flags.name == "" {
				flags.name = args[0]
			}

			cancelRequest.Organization = ch.Config.Organization
			cancelRequest.Database = flags.name

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !flags.force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf("cannot cancel import with the output format %q (run with -force to override)", ch.Printer.Format())
				}

				confirmationName := fmt.Sprintf("%s/%s", cancelRequest.Organization, cancelRequest.Database)
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm cancellation of import %q (run with -force to override)", confirmationName)
				}

				confirmationMessage := fmt.Sprintf("%s %s %s", printer.Bold("Please type"), printer.BoldBlue(confirmationName), printer.Bold("to confirm:"))

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
					return errors.New("incorrect import name entered, skipping import cancellation")
				}
			}

			err = client.DataImports.CancelDataImport(ctx, cancelRequest)
			if err != nil {
				return err
			}

			ch.Printer.Printf("Data import into database %v has been canceleed", flags.name)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Cancel an import without confirmation")

	return cmd
}
