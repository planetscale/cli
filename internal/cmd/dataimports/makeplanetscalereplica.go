package dataimports

import (
	"fmt"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func MakePlanetScaleReplicaCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name  string
		force bool
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

			if !flags.force {
				confirmationName := fmt.Sprintf("%s/%s", makeReplicaReq.Organization, makeReplicaReq.Database)
				confirmError := ch.Printer.ConfirmCommand(confirmationName, "make replica", "demotion to replica")
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
					return fmt.Errorf("unable to switch PlanetScale database %s to Primary", flags.name)
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if dataImport.ImportState == ps.DataImportReady {
				return fmt.Errorf("cannot make PlanetScale Database %s/%s Replica because this import has completed", getImportReq.Organization, getImportReq.Database)
			}

			if dataImport.ImportState != ps.DataImportSwitchTrafficCompleted {
				return fmt.Errorf("cannot make PlanetScale Database %s/%s Replica because it is not serving as a Primary", getImportReq.Organization, getImportReq.Database)
			}

			end = ch.Printer.PrintProgress(fmt.Sprintf("Switching PlanetScale database %s to Primary...", printer.BoldBlue(flags.name)))
			defer end()
			dataImport, err = client.DataImports.MakePlanetScaleReplica(ctx, makeReplicaReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("unable to switch PlanetScale database %s to Replica", flags.name)
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()
			ch.Printer.Printf("Successfully switch PlanetScale database %s to Replica.\n", printer.BoldBlue(flags.name))
			PrintDataImport(ch.Printer, *dataImport)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "PlanetScale database importing data")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Make PlanetScale database Replica without confirmation")
	cmd.MarkPersistentFlagRequired("name")

	return cmd
}
