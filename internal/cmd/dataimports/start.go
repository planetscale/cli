package dataimports

import (
	"errors"
	"fmt"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
	"strings"
)

func StartDataImportCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name     string
		host     string
		username string
		password string
		database string
		port     int
		dryRun   bool
	}

	startImportRequest := &ps.StartDataImportRequest{}
	testRequest := &ps.TestDataImportSourceRequest{}
	cmd := &cobra.Command{
		Use:     "start [options]",
		Short:   "start importing data from an external database",
		Aliases: []string{"s"},
		RunE: func(cmd *cobra.Command, args []string) error {

			ctx := cmd.Context()
			dataSource := ps.DataImportSource{
				Database:            flags.database,
				UserName:            flags.username,
				Password:            flags.password,
				HostName:            flags.host,
				Port:                flags.port,
				SSLVerificationMode: ps.SSLModeDisabled,
			}
			startImportRequest.Organization = ch.Config.Organization
			startImportRequest.DatabaseName = flags.name
			startImportRequest.Source = dataSource

			testRequest.Organization = ch.Config.Organization
			testRequest.Database = flags.database
			testRequest.Source = dataSource

			client, err := ch.Client()
			if err != nil {
				return err
			}
			ch.Printer.Println(fmt.Sprintf("testing Compatibility of external database %s...", printer.BoldBlue(flags.database)))

			resp, err := client.DataImports.TestDataImportSource(ctx, testRequest)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("unable to check compatibility of database %s, hosted at %s", flags.database, flags.host)
				default:
					return cmdutil.HandleError(err)
				}
			}

			if !resp.CanConnect && len(resp.ConnectError) > 0 {
				return errors.New(resp.ConnectError)
			}

			if len(resp.Errors) > 0 {
				var sb strings.Builder
				sb.WriteString(printer.Red("External database compatibility check failed. "))
				sb.WriteString("Fix the following errors and then try again:\n\n")
				for _, compatError := range resp.Errors {
					fmt.Fprintf(&sb, "• %s\n", compatError.ErrorDescription)
				}

				return errors.New(sb.String())
			}
			ch.Printer.Printf("external database %s is compatible and can be imported into PlanetScale database %s\n", printer.BoldBlue(flags.database), printer.BoldGreen(flags.name))
			if flags.dryRun {
				return nil
			}

			ch.Printer.Println(fmt.Sprintf("starting import of schema and data from external database %s to PlanetScale database %s", printer.BoldBlue(flags.database), printer.BoldGreen(flags.name)))
			dataImport, err := client.DataImports.StartDataImport(ctx, startImportRequest)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("unable to check compatibility of database %s, hosted at %s", flags.database, flags.host)
				default:
					return cmdutil.HandleError(err)
				}
			}

			ch.Printer.Printf("database %s hosted at %s is being imported into PlanetScale database %s\n", flags.database, flags.host, flags.name)
			ch.Printer.PrintDataImport(*dataImport)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "")
	cmd.PersistentFlags().StringVar(&flags.host, "host", "", "")
	cmd.PersistentFlags().StringVar(&flags.database, "database", "", "")
	cmd.PersistentFlags().StringVar(&flags.username, "username", "", "")
	cmd.PersistentFlags().StringVar(&flags.password, "password", "", "")
	cmd.PersistentFlags().IntVar(&flags.port, "port", 3306, "")
	cmd.PersistentFlags().BoolVar(&flags.dryRun, "dry-run", true, "only run compatibility check, do not start import")

	return cmd
}
