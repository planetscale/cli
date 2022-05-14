package dataimports

import (
	"errors"
	"fmt"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func GetDataImportCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name string
	}

	getRequest := &ps.GetImportStatusRequest{}

	cmd := &cobra.Command{
		Use:     "get [database]",
		Short:   "get the current state of a data import request into a PlanetScale database",
		Aliases: []string{"g"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			getRequest.Organization = ch.Config.Organization
			getRequest.Database = flags.name

			client, err := ch.Client()
			if err != nil {
				return err
			}

			//end := ch.Printer.PrintProgress(fmt.Sprintf("Testing Compatibility of database %s with user %s...", printer.BoldBlue(flags.database), printer.BoldBlue(flags.username)))
			//defer end()

			resp, err := client.DataImports.GetDataImportStatus(ctx, getRequest)
			if err != nil {
				return err
			}

			ch.Printer.PrintDataImport(*resp)

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

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "PlanetScale database importing data")
	cmd.MarkPersistentFlagRequired("name")

	return cmd
}
