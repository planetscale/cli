package dataimports

import (
	"fmt"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func DetachExternalDatabaseCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name string
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

			dataImport, err := client.DataImports.DetachExternalDatabase(ctx, detachExternalDatabaseReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("unable to detach external database for PlanetScale database %s", flags.name)
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
