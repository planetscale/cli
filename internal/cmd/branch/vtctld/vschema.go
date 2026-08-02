package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/spf13/cobra"
)

// GetVSchemaCmd reads the live VSchema for a keyspace from the cluster via vtctld.
func GetVSchemaCmd(ch *cmdutil.Helper) *cobra.Command {
	var keyspace string

	cmd := &cobra.Command{
		Use:   "get-vschema <database> <branch>",
		Short: "Get the live VSchema for a keyspace",
		Long: "Get the live VSchema for a keyspace from the cluster via vtctld. " +
			"This reads the current cluster state, unlike `pscale keyspace vschema show`, " +
			"which reads from the schema snapshot.",
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Fetching VSchema for keyspace %s on %s\u2026",
					keyspace, progressTarget(ch.Config.Organization, database, branch)))
			defer end()

			data, err := client.Vtctld.GetVSchema(ctx, &ps.VtctldGetVSchemaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     keyspace,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&keyspace, "keyspace", "", "Keyspace name")
	cmd.MarkFlagRequired("keyspace")

	return cmd
}
