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

	cmd := &cobra.Command{
		Use:   "create <database>",
		Short: "Create a database instance",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			clusterSize, err := cmd.Flags().GetString("cluster-size")
			if err != nil {
				return err
			}

			createReq.ClusterSize = clusterSize

			createReq.Organization = ch.Config.Organization
			createReq.Name = args[0]

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

	cmd.Flags().StringVar(&createReq.Notes, "notes", "", "notes for the database")
	cmd.Flags().MarkDeprecated("notes", "is no longer available.")
	cmd.Flags().StringVar(&createReq.Region, "region", "", "region for the database")

	cmd.Flags().String("cluster-size", "PS-10", "cluster size for Scaler Pro databases. Use `pscale size cluster list` to see the valid sizes.")

	cmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.RegionsCompletionFunc(ch, cmd, args, toComplete)
	})

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.ClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}
