package dataimports

import (
	"errors"
	"fmt"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CancelDataImportCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name string
	}

	cancelRequest := &ps.CancelDataImportRequest{}

	cmd := &cobra.Command{
		Use:     "get [database]",
		Short:   "get the current state of a data import request into a planetscale database",
		Aliases: []string{"g"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cancelRequest.Organization = ch.Config.Organization
			cancelRequest.Database = flags.name

			client, err := ch.Client()
			if err != nil {
				return err
			}

			resp, err := client.DataImports.CancelDataImport(ctx, cancelRequest)
			if err != nil {
				return err
			}

			completedSteps := printer.GetCompletedImportStates(resp.ImportState)
			if len(completedSteps) > 0 {
				ch.Printer.Println(completedSteps)
			}

			inProgressStep, _ := printer.GetCurrentImportState(resp.ImportState)
			if len(inProgressStep) > 0 {
				ch.Printer.Println(inProgressStep)
			}

			pendingSteps := printer.GetPendingImportStates(resp.ImportState)
			if len(pendingSteps) > 0 {
				ch.Printer.Println(pendingSteps)
			}

			if resp.ImportState == ps.DataImportPreparingDataCopyFailed ||
				resp.ImportState == ps.DataImportCopyingDataFailed ||
				resp.ImportState == ps.DataImportSwitchTrafficError ||
				resp.ImportState == ps.DataImportReverseTrafficError ||
				resp.ImportState == ps.DataImportDetachExternalDatabaseError {
				return errors.New(fmt.Sprintf("import from external database into PlanetScale failed with \n %s \n current state is %s", printer.BoldRed(resp.Errors), resp.ImportState))
			}

			if resp.ImportState == ps.DataImportSwitchTrafficPending {
				ch.Printer.Printf("all data and schema has been copied from the external database and your PlanetScale database %s is running in replica mode\n", printer.BoldGreen(flags.name))
				ch.Printer.Printf("you should now be able to switch your PlanetScale database %s into primary mode using the \"make-primary\" command \n", printer.BoldGreen(flags.name))
			}

			if resp.ImportState == ps.DataImportSwitchTrafficCompleted {
				ch.Printer.Printf("Your PlanetScale database %s is now running as a primary \n", printer.BoldGreen(flags.name))
				ch.Printer.Printf("if necessary, you can use  the \"make-replica\" command to switch back to replica mode\n")
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "")

	return cmd
}
