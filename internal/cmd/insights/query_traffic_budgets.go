package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmd/trafficcontrol"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func QueryTrafficBudgetsCmd(ch *cmdutil.Helper) *cobra.Command {
	var keyspace string

	cmd := &cobra.Command{
		Use:     "traffic-budgets <database> <branch> <fingerprint>",
		Short:   "List traffic budgets affecting a query fingerprint",
		Aliases: []string{"ls"},
		Args:    cmdutil.RequiredArgs("database", "branch", "fingerprint"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, fingerprint := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching traffic budgets for fingerprint %s on %s/%s...",
				printer.BoldBlue(fingerprint), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			budgets, err := client.QueryInsights.ListQueryTrafficBudgets(ctx, &ps.ListQueryTrafficBudgetsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Fingerprint:  fingerprint,
			}, ps.WithKeyspace(keyspace))
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}
			end()

			if len(budgets) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No traffic budgets affect fingerprint %s on %s/%s.\n",
					printer.BoldBlue(fingerprint), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(trafficcontrol.ToTrafficBudgets(budgets))
		},
	}

	cmd.Flags().StringVar(&keyspace, "keyspace", "", "Keyspace for the fingerprint")

	return cmd
}
