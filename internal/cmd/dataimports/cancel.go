package dataimports

import (
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CancelDataImportCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name string
	}

	cancelRequest := &ps.CancelDataImportRequest{}

	cmd := &cobra.Command{
		Use:     "cancel [database]",
		Short:   "cancel data import request into a planetscale database",
		Aliases: []string{"c"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cancelRequest.Organization = ch.Config.Organization
			cancelRequest.Database = flags.name

			client, err := ch.Client()
			if err != nil {
				return err
			}

			dataImport, err := client.DataImports.CancelDataImport(ctx, cancelRequest)
			if err != nil {
				return err
			}

			ch.Printer.PrintDataImport(*dataImport)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "")

	return cmd
}
