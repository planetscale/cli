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

func CancelDataImportCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		host     string
		username string
		password string
		database string
		port     int
	}

	testRequest := &ps.TestDataImportSourceRequest{}

	cmd := &cobra.Command{
		Use:     "cancel [import-id]",
		Short:   "Cancel an existing data import request",
		Aliases: []string{"g"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			testRequest.Organization = ch.Config.Organization
			testRequest.Source = ps.DataImportSource{
				Database: flags.database,
				UserName: flags.username,
				Password: flags.password,
				HostName: flags.host,
				Port:     flags.port,
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

			if len(resp.Errors) > 0 {

				var sb strings.Builder
				sb.WriteString(printer.Red("Branch promotion failed. "))
				sb.WriteString("Fix the following errors and then try again:\n\n")
				for _, compatError := range resp.Errors {
					fmt.Fprintf(&sb, "• %s\n", compatError.ErrorDescription)
				}

				return errors.New(sb.String())
			}

			end()

			ch.Printer.Printf("Database %s hosted at %s is compatible and can be imported into PlanetScale ", flags.database, flags.host)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.host, "host", "", "")
	cmd.PersistentFlags().StringVar(&flags.database, "database", "", "")
	cmd.PersistentFlags().StringVar(&flags.username, "username", "", "")
	cmd.PersistentFlags().StringVar(&flags.password, "password", "", "")
	cmd.PersistentFlags().IntVar(&flags.port, "port", 3306, "")

	return cmd
}
