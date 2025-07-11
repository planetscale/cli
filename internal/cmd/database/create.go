package database

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// CreateCmd is the command for creating a database.
func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &ps.CreateDatabaseRequest{}

	var flags struct {
		clusterSize string
		engine      string
	}

	cmd := &cobra.Command{
		Use:   "create <database>",
		Short: "Create a database instance",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			createReq.Organization = ch.Config.Organization
			createReq.Name = args[0]

			engine, err := parseDatabaseEngine(flags.engine)
			if err != nil {
				return err
			}

			createReq.Kind = engine

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress("Creating database...")
			defer end()
			database, err := client.Databases.Create(ctx, createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("organization %s does not exist", printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Database %s was successfully created.\n\nView this database in the browser: %s\n", printer.BoldBlue(database.Name), printer.BoldBlue(database.HtmlURL))
				return nil
			}

			return ch.Printer.PrintResource(toDatabase(database))
		},
	}

	cmd.Flags().StringVar(&createReq.Region, "region", "", "region for the database")

	cmd.Flags().StringVar(&createReq.ClusterSize, "cluster-size", "", "cluster size for Scaler Pro databases. Use `pscale size cluster list` to see the valid sizes.")
	cmd.Flags().StringVar(&flags.engine, "engine", string(ps.DatabaseEngineMySQL), "The database engine for the database. Supported values: mysql, postgresql. Defaults to mysql.")
	cmd.RegisterFlagCompletionFunc("engine", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return []cobra.Completion{
			cobra.CompletionWithDesc("mysql", "A Vitess database"),
			cobra.CompletionWithDesc("postgresql", "The fastest cloud Postgres"),
		}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.RegionsCompletionFunc(ch, cmd, args, toComplete)
	})

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.ClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}

func parseDatabaseEngine(engine string) (ps.DatabaseEngine, error) {
	switch engine {
	case "mysql":
		return ps.DatabaseEngineMySQL, nil
	case "postgresql", "postgres":
		return ps.DatabaseEnginePostgres, nil
	default:
		return ps.DatabaseEngineMySQL, fmt.Errorf("invalid database engine %q, supported values: mysql, postgresql", engine)
	}
}
