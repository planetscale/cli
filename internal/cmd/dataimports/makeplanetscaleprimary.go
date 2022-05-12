package dataimports

import (
	"fmt"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func MakePlanetScalePrimaryCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name string
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

			getImportReq := &ps.GetImportStatusRequest{
				Organization: ch.Config.Organization,
				Database:     flags.name,
			}

			dataImport, err := client.DataImports.GetDataImportStatus(ctx, getImportReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("unable to switch PlanetScale database %s to primary", flags.name)
				default:
					return cmdutil.HandleError(err)
				}
			}

			if dataImport.ImportState != ps.DataImportSwitchTrafficPending {
				return fmt.Errorf("cannot make PlanetScale Database %s/%s Primary because it is not serving as a Replica", getImportReq.Organization, getImportReq.Database)
			}

			dataImport, err = client.DataImports.MakePlanetScalePrimary(ctx, makePrimaryReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("unable to switch PlanetScale database %s to primary", flags.name)
				default:
					return cmdutil.HandleError(err)
				}
			}

			ch.Printer.PrintDataImport(*dataImport)

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "")

	return cmd
}
