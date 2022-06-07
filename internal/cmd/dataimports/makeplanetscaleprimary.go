package dataimports

import (
	"fmt"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func MakePlanetScalePrimaryCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name  string
		force bool
	}

	makePrimaryReq := &ps.MakePlanetScalePrimaryRequest{}

	cmd := &cobra.Command{
		Use:     "make-primary [options]",
		Short:   "mark PlanetScale's database as the Primary, and the external database as Replica",
		Aliases: []string{"mp"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			makePrimaryReq.Organization = ch.Config.Organization
			makePrimaryReq.Database = flags.name

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !flags.force {
				confirmationName := fmt.Sprintf("%s/%s", makePrimaryReq.Organization, makePrimaryReq.Database)
				confirmError := ch.Printer.ConfirmCommand(confirmationName, "make primary", "promotion to primary")
				if confirmError != nil {
					return confirmError
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Getting current import status for PlanetScale database %s...", printer.BoldBlue(flags.name)))
			defer end()
			getImportReq := &ps.GetImportStatusRequest{
				Organization: ch.Config.Organization,
				Database:     flags.name,
			}

			dataImport, err := client.DataImports.GetDataImportStatus(ctx, getImportReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("PlanetScale database %s is not importing data", flags.name)
				default:
					return cmdutil.HandleError(err)
				}
			}

			if dataImport.ImportState != ps.DataImportSwitchTrafficPending {
				reason := "it is already switched to Primary"
				switch dataImport.ImportState {
				case ps.DataImportCopyingData, ps.DataImportPreparingDataCopy:
					reason = "we are still copying data from upstream database"
				case ps.DataImportCopyingDataFailed, ps.DataImportPreparingDataCopyFailed:
					reason = "we are unable to copy data from upstream database"
				case ps.DataImportReady:
					reason = "this import has completed"
				}
				return fmt.Errorf("cannot make PlanetScale Database %s/%s Primary because %s", getImportReq.Organization, getImportReq.Database, reason)
			}
			end()
			end = ch.Printer.PrintProgress(fmt.Sprintf("Switching PlanetScale database %s to Primary...", printer.BoldBlue(flags.name)))
			defer end()

			dataImport, err = client.DataImports.MakePlanetScalePrimary(ctx, makePrimaryReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("unable to switch PlanetScale database %s to Primary", flags.name)
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			ch.Printer.Printf("Successfully switch PlanetScale database %s to Primary.\n", printer.BoldBlue(flags.name))
			PrintDataImport(ch.Printer, *dataImport)

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "PlanetScale database importing data")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Make PlanetScale database Primary without confirmation")
	cmd.MarkPersistentFlagRequired("name")

	return cmd
}
