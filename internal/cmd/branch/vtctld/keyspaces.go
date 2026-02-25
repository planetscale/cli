package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ListKeyspacesCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name string
	}

	cmd := &cobra.Command{
		Use:   "list-keyspaces <database> <branch>",
		Short: "List vtctld keyspaces for a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Fetching keyspaces for %s/%s\u2026",
					printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.Vtctld.ListKeyspaces(ctx, &ps.VtctldListKeyspacesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Name:         flags.name,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Filter by keyspace name")

	return cmd
}
