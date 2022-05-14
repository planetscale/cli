package dataimports

import (
	"fmt"
	"github.com/planetscale/cli/internal/cmdutil"
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

			err = client.DataImports.CancelDataImport(ctx, cancelRequest)
			if err != nil {
				return err
			}

			ch.Printer.Printf("Data import into database %v has been canceleed", flags.name)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "PlanetScale database importing data")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Cancel an import without confirmation")
	cmd.MarkPersistentFlagRequired("name")

	return cmd
}
