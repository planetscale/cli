package keyspace

import (
	"fmt"
	"os"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func CreateExternalCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &planetscale.CreateExternalKeyspaceRequest{}

	var flags struct {
		host           string
		sourceDatabase string
		username       string
		password       string
		port           int
		sslMode        string
		sslCA          string
		sslKey         string
		sslCertificate string
		sslServerName  string
		minTLSVersion  string
		tabletCell     string
		clusterSize    string
		skipLintErrors bool
		dryRun         bool
		wait           bool
	}

	cmd := &cobra.Command{
		Use:   "create-external <database> <branch> <keyspace>",
		Short: "Create an external keyspace on a branch",
		Long: `Create an external keyspace by connecting a branch to an existing MySQL database.

Connection flags follow pscale data-imports start. --source-database is the
remote MySQL database name, not the PlanetScale database. --cluster-size is
optional and selects the external tablet size; when omitted, PlanetScale
chooses a size from the source storage. Managed organizations should pass a
size from pscale size cluster list.`,
		Args: cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			sslCA, err := readFlagFileOrString(flags.sslCA)
			if err != nil {
				return err
			}
			sslCert, err := readFlagFileOrString(flags.sslCertificate)
			if err != nil {
				return err
			}
			sslKey, err := readFlagFileOrString(flags.sslKey)
			if err != nil {
				return err
			}

			datasource := planetscale.ExternalDatasource{
				DatabaseName:  flags.sourceDatabase,
				Hostname:      flags.host,
				Port:          flags.port,
				Username:      flags.username,
				Password:      flags.password,
				SSLMode:       cmdutil.ParseSSLMode(flags.sslMode).String(),
				SSLCA:         sslCA,
				SSLCert:       sslCert,
				SSLKey:        sslKey,
				SSLServerName: flags.sslServerName,
				MinTLSVersion: flags.minTLSVersion,
				TabletCell:    flags.tabletCell,
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if flags.dryRun {
				end := ch.Printer.PrintProgress(fmt.Sprintf("Checking compatibility of %s for %s/%s", printer.BoldBlue(flags.sourceDatabase), printer.BoldBlue(database), printer.BoldBlue(branch)))
				defer end()

				resp, err := client.Keyspaces.LintExternal(ctx, &planetscale.LintExternalKeyspaceRequest{
					Organization:       ch.Config.Organization,
					Database:           database,
					Branch:             branch,
					ExternalDatasource: datasource,
				})
				if err != nil {
					switch cmdutil.ErrCode(err) {
					case planetscale.ErrNotFound:
						return fmt.Errorf("database %s or branch %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
					default:
						return cmdutil.HandleError(err)
					}
				}
				end()

				if !resp.CanConnect || resp.Error != "" {
					if resp.Error != "" {
						return fmt.Errorf("%s", resp.Error)
					}
					return fmt.Errorf("unable to connect to %s", flags.host)
				}

				if ch.Printer.Format() == printer.Human {
					if len(resp.LintErrors) > 0 {
						ch.Printer.Printf("External database %s can be reached, but reported %d schema lint error(s):\n", printer.BoldBlue(flags.sourceDatabase), len(resp.LintErrors))
						for _, lintError := range resp.LintErrors {
							ch.Printer.Printf("  %s: %s\n", printer.BoldRed(lintError.TableName), lintError.ErrorDescription)
						}
						return nil
					}

					ch.Printer.Printf("External database %s is compatible with keyspace %s.\n", printer.BoldBlue(flags.sourceDatabase), printer.BoldBlue(keyspace))
					return nil
				}

				return ch.Printer.PrintResource(resp)
			}

			createReq.Organization = ch.Config.Organization
			createReq.Database = database
			createReq.Branch = branch
			createReq.Name = keyspace
			createReq.ClusterSize = flags.clusterSize
			createReq.SkipLintErrors = flags.skipLintErrors
			createReq.ExternalDatasource = datasource

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating external keyspace %s in %s/%s", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			k, err := client.Keyspaces.CreateExternal(ctx, createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s or branch %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if flags.wait {
				end := ch.Printer.PrintProgress(fmt.Sprintf("Waiting until keyspace %s is ready...", printer.BoldBlue(keyspace)))
				defer end()

				k, err = waitUntilReady(ctx, client, ch.Printer, ch.Debug(), &planetscale.GetKeyspaceRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Branch:       branch,
					Keyspace:     keyspace,
				})
				if err != nil {
					return err
				}
				end()
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("External keyspace %s was successfully created.\n", printer.BoldBlue(k.Name))
				return nil
			}

			return ch.Printer.PrintResource(toKeyspace(k))
		},
	}

	cmd.Flags().StringVar(&flags.host, "host", "", "Host name of the external database")
	cmd.Flags().StringVar(&flags.sourceDatabase, "source-database", "", "Name of the database on the external MySQL server")
	cmd.Flags().StringVar(&flags.username, "username", "", "Username to connect to the external database")
	cmd.Flags().StringVar(&flags.password, "password", "", "Password to connect to the external database")
	cmd.Flags().IntVar(&flags.port, "port", 3306, "Port number to connect to the external database")
	cmd.Flags().StringVar(&flags.sslMode, "ssl-mode", "", "SSL verification mode, allowed values: disabled, preferred, required, verify_ca, verify_identity")
	cmd.Flags().StringVar(&flags.sslCA, "ssl-certificate-authority", "", "CA certificate chain, or a path to a PEM file")
	cmd.Flags().StringVar(&flags.sslKey, "ssl-client-key", "", "Client private key, or a path to a PEM file")
	cmd.Flags().StringVar(&flags.sslCertificate, "ssl-client-certificate", "", "Client certificate, or a path to a PEM file")
	cmd.Flags().StringVar(&flags.sslServerName, "ssl-server-name", "", "SSL server name override")
	cmd.Flags().StringVar(&flags.minTLSVersion, "min-tls-version", "", "Minimum TLS version")
	cmd.Flags().StringVar(&flags.tabletCell, "tablet-cell", "", "Cell where the external tablet runs")
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "External tablet size. Optional; defaults from source storage. Use `pscale size cluster list` for valid sizes.")
	cmd.Flags().BoolVar(&flags.skipLintErrors, "skip-lint-errors", false, "Create even if datasource lint reports errors, when the organization allows it")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Check compatibility with the external database without creating the keyspace")
	cmd.Flags().BoolVar(&flags.wait, "wait", false, "Wait until the keyspace is ready")

	cmd.MarkFlagRequired("host")
	cmd.MarkFlagRequired("source-database")
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("password")
	cmd.MarkFlagRequired("ssl-mode")

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.ExternalClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}

func readFlagFileOrString(value string) (string, error) {
	if value == "" || strings.Contains(value, "\n") {
		return value, nil
	}

	info, err := os.Stat(value)
	if err != nil {
		return value, nil
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", value)
	}

	b, err := os.ReadFile(value)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
