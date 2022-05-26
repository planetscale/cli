package dataimports

import (
	"fmt"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
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
				confirmationName := fmt.Sprintf("%s/%s", cancelRequest.Organization, cancelRequest.Database)
				confirmError := ch.Printer.ConfirmCommand(confirmationName, "cancel import", "cancellation of import")
				if confirmError != nil {
					return confirmError
				}
			}

			getImportReq := &ps.GetImportStatusRequest{
				Organization: ch.Config.Organization,
				Database:     flags.name,
			}
			end := ch.Printer.PrintProgress(fmt.Sprintf("Getting current import status for PlanetScale database %s...", printer.BoldBlue(flags.name)))
			defer end()

			dataImport, err := client.DataImports.GetDataImportStatus(ctx, getImportReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("unable to cancel import into PlanetScale database %s", flags.name)
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if dataImport.ImportState == ps.DataImportReady {
				return fmt.Errorf("cannot cancel import into PlanetScale Database %s/%s because this import has completed", getImportReq.Organization, getImportReq.Database)
			}

			end = ch.Printer.PrintProgress(fmt.Sprintf("Cancelling Data Import into PlanetScale database %s...", printer.BoldBlue(getImportReq.Organization+"/"+flags.name)))
			defer end()

			err = client.DataImports.CancelDataImport(ctx, cancelRequest)
			if err != nil {
				return err
			}
			end()
			ch.Printer.Printf("Data Import into PlanetScale database %v/%v has been cancelled", getImportReq.Organization, flags.name)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "PlanetScale database importing data")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Cancel an import without confirmation")
	cmd.MarkPersistentFlagRequired("name")

	return cmd
}
