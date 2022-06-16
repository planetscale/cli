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

func LintExternalDataSourceCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		host     string
		username string
		password string
		database string
		port     int
		sslMode  string
	}

	testRequest := &ps.TestDataImportSourceRequest{}

	cmd := &cobra.Command{
		Use:     "lint [options]",
		Short:   "lint external database for compatibility with PlanetScale",
		Aliases: []string{"l"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			sslMode := cmdutil.ParseSSLMode(flags.sslMode)
			testRequest.Organization = ch.Config.Organization
			testRequest.Database = flags.database
			testRequest.Connection = ps.DataImportSource{
				Database:            flags.database,
				UserName:            flags.username,
				Password:            flags.password,
				HostName:            flags.host,
				Port:                flags.port,
				SSLVerificationMode: sslMode,
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Testing Compatibility of database %s with user %s...", printer.BoldBlue(flags.database), printer.BoldBlue(flags.username)))
			defer end()

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
				sb.WriteString(printer.Red("External database compatibility check failed.\n"))
				sb.WriteString("Please fix the following errors and then try again:\n")

				for idx, compatError := range resp.Errors {
					fmt.Fprintf(&sb, "%v. %s\n", (idx + 1), compatError.ErrorDescription)
				}

				return errors.New(sb.String())
			}
			end()

			ch.Printer.Printf("Database %s hosted at %s is compatible and can be imported into PlanetScale!\n", flags.database, flags.host)
			if resp.SuggestedBillingPlan == ps.ScalerPlan {
				ch.Printer.Printf("\nIf you choose to continue, the imported database will be on Scaler plan. The monthly cost is %v.\n", printer.BoldYellow("$29"))
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.host, "host", "", "Host name of the external database.")
	cmd.PersistentFlags().StringVar(&flags.database, "database", "", "Name of the external database")
	cmd.PersistentFlags().StringVar(&flags.username, "username", "", "Username to connect to external database.")
	cmd.PersistentFlags().StringVar(&flags.password, "password", "", "Password to connect to external database.")
	cmd.PersistentFlags().StringVar(&flags.sslMode, "ssl-mode", "", "SSL verification mode, allowed values : disabled, preferred, required, verify_ca, verify_identity")
	cmd.PersistentFlags().IntVar(&flags.port, "port", 3306, "Port number to connect to external database")

	cmd.MarkPersistentFlagRequired("host")
	cmd.MarkPersistentFlagRequired("database")
	cmd.MarkPersistentFlagRequired("username")
	cmd.MarkPersistentFlagRequired("password")
	cmd.MarkPersistentFlagRequired("ssl-mode")

	return cmd
}
