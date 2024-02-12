package dataimports

import (
	"errors"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func StartDataImportCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name           string
		host           string
		region         string
		username       string
		password       string
		database       string
		port           int
		dryRun         bool
		sslMode        string
		sslCA          string
		sslKey         string
		sslCertificate string
		sslServerName  string
	}

	startImportRequest := &ps.StartDataImportRequest{}
	testRequest := &ps.TestDataImportSourceRequest{}
	cmd := &cobra.Command{
		Use:     "start [options]",
		Short:   "start importing data from an external database",
		Aliases: []string{"s"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sslMode := cmdutil.ParseSSLMode(flags.sslMode)

			dataSource := ps.DataImportSource{
				Database:            flags.database,
				UserName:            flags.username,
				Password:            flags.password,
				HostName:            flags.host,
				Port:                flags.port,
				SSLVerificationMode: sslMode,
				SSLKey:              flags.sslKey,
				SSLCertificate:      flags.sslCertificate,
				SSLCA:               flags.sslCA,
				SSLServerName:       flags.sslServerName,
			}
			startImportRequest.Organization = ch.Config.Organization
			startImportRequest.Database = flags.name
			startImportRequest.Connection = dataSource
			startImportRequest.Region = flags.region

			testRequest.Organization = ch.Config.Organization
			testRequest.Database = flags.database
			testRequest.Connection = dataSource

			client, err := ch.Client()
			if err != nil {
				return err
			}
			ch.Printer.Println(fmt.Sprintf("Testing Compatibility of database %s with user %s...", printer.BoldBlue(flags.database), printer.BoldYellow(flags.username)))

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
			ch.Printer.Printf("Database %s is compatible and can be imported into PlanetScale database %s\n", printer.BoldBlue(flags.database), printer.BoldGreen(flags.name))
			if resp.SuggestedBillingPlan == ps.ScalerProPlan {
				ch.Printer.Println("If you choose to continue, the imported database will be on Scaler Pro with a PS-10.")
			}
			if flags.dryRun {
				ch.Printer.Println("Please run this command with --dry-run=false to start the import")
				return nil
			}
			if resp.SuggestedBillingPlan == ps.ScalerProPlan {
				confirmationName := "start"
				confirmError := ch.Printer.ConfirmCommand(confirmationName, "import", "import into PlanetScale")
				if confirmError != nil {
					return confirmError
				}
			}

			startImportRequest.Plan = resp.SuggestedPlan
			startImportRequest.MaxPoolSize = resp.MaxPoolSize

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
			PrintDataImport(ch.Printer, *dataImport)

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.name, "name", "", "")
	cmd.PersistentFlags().StringVar(&flags.region, "region", "", "region for the PlanetScale database.")
	cmd.PersistentFlags().StringVar(&flags.host, "host", "", "Host name of the external database.")
	cmd.PersistentFlags().StringVar(&flags.database, "database", "", "Name of the external database")
	cmd.PersistentFlags().StringVar(&flags.username, "username", "", "Username to connect to external database.")
	cmd.PersistentFlags().StringVar(&flags.password, "password", "", "Password to connect to external database.")
	cmd.PersistentFlags().IntVar(&flags.port, "port", 3306, "Port number to connect to external database")
	cmd.PersistentFlags().BoolVar(&flags.dryRun, "dry-run", true, "Only run compatibility check, do not start import")
	cmd.PersistentFlags().StringVar(&flags.sslMode, "ssl-mode", "", "SSL verification mode, allowed values: disabled, preferred, required, verify_ca, verify_identity")
	cmd.PersistentFlags().StringVar(&flags.sslServerName, "ssl-server-name", "", "SSL server name override")
	cmd.PersistentFlags().StringVar(&flags.sslCA, "ssl-certificate-authority", "", "Provide the full CA certificate chain here")
	cmd.PersistentFlags().StringVar(&flags.sslKey, "ssl-client-key", "", "Private key for the client certificate")
	cmd.PersistentFlags().StringVar(&flags.sslKey, "ssl-client-certificate", "", "Client Certificate to authenticate PlanetScale with your database server")

	cmd.MarkPersistentFlagRequired("name")
	cmd.MarkPersistentFlagRequired("host")
	cmd.MarkPersistentFlagRequired("database")
	cmd.MarkPersistentFlagRequired("username")
	cmd.MarkPersistentFlagRequired("password")
	cmd.MarkPersistentFlagRequired("ssl-mode")

	return cmd
}
