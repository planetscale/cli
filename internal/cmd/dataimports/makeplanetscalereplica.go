package dataimports

import (
	"fmt"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func MakePlanetScaleReplicaCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name string
	}

	makeReplicaReq := &ps.MakePlanetScaleReplicaRequest{}

	cmd := &cobra.Command{
		Use:     "make-replica [options]",
		Short:   "mark PlanetScale's database as the Replica, and the external database as Primary",
		Aliases: []string{"mr"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			makeReplicaReq.Organization = ch.Config.Organization
			makeReplicaReq.Database = flags.name

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

			if dataImport.ImportState != ps.DataImportSwitchTrafficCompleted {
				return fmt.Errorf("cannot make PlanetScale Database %s/%s Replica because it is not serving as a Primary", getImportReq.Organization, getImportReq.Database)
			}

			dataImport, err = client.DataImports.MakePlanetScaleReplica(ctx, makeReplicaReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("unable to switch PlanetScale database %s to Replica", flags.name)
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
