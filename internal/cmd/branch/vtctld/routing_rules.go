package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// GetRoutingRulesCmd reads live routing rules from the cluster via vtctld.
func GetRoutingRulesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-routing-rules <database> <branch>",
		Short: "Get live routing rules for a branch",
		Long: "Get live routing rules from the cluster via vtctld. " +
			"This reads the current cluster state, unlike `pscale branch routing-rules get`, " +
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
				fmt.Sprintf("Fetching routing rules for %s\u2026",
					progressTarget(ch.Config.Organization, database, branch)))
			defer end()

			data, err := client.Vtctld.GetRoutingRules(ctx, &ps.VtctldGetRoutingRulesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	return cmd
}
