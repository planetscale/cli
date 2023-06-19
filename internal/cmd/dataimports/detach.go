package dataimports

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func DetachExternalDatabaseCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name  string
		force bool
	}

	detachExternalDatabaseReq := &ps.DetachExternalDatabaseRequest{}

	cmd := &cobra.Command{
		Use:     "detach-external-database [options]",
		Short:   "detach external database that is used as a source for PlanetScale database",
		Aliases: []string{"ded"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			detachExternalDatabaseReq.Organization = ch.Config.Organization
			detachExternalDatabaseReq.Database = flags.name

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !flags.force {
				confirmationName := fmt.Sprintf("%s/%s", detachExternalDatabaseReq.Organization, detachExternalDatabaseReq.Database)
				confirmError := ch.Printer.ConfirmCommand(confirmationName, "detach external database", "detaching external database")
				if confirmError != nil {
					return confirmError
				}
			}

			getImportReq := &ps.GetImportStatusRequest{
				Organization: ch.Config.Organization,
				Database:     flags.name,
			}

			dataImport, err := client.DataImports.GetDataImportStatus(ctx, getImportReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("unable to switch PlanetScale database %s to Primary", flags.name)
				default:
					return cmdutil.HandleError(err)
				}
			}

			if dataImport.ImportState != ps.DataImportSwitchTrafficCompleted {
				return fmt.Errorf("cannot detach external database %s at %s because PlanetScale is not serving as a Primary", getImportReq.Organization, getImportReq.Database)
			}

			dataImport, err = client.DataImports.DetachExternalDatabase(ctx, detachExternalDatabaseReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("unable to detach external database for PlanetScale database %s", flags.name)
				default:
					return cmdutil.HandleError(err)
				}
			}

			PrintDataImport(ch.Printer, *dataImport)

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "PlanetScale database importing data")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Make PlanetScale database Replica without confirmation")
	cmd.MarkPersistentFlagRequired("name")
	return cmd
}
