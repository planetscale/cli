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
	var flags struct {
		keyspace string
		page     int
		perPage  int
	}

	cmd := &cobra.Command{
		Use:   "traffic-budgets <database> <branch> <fingerprint>",
		Short: "List traffic budgets affecting a query fingerprint",
		Args:  cmdutil.RequiredArgs("database", "branch", "fingerprint"),
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
			}, ps.WithKeyspace(flags.keyspace), ps.WithPage(flags.page), ps.WithPerPage(flags.perPage))
			if err != nil {
				return notFoundError(ch, err, database, branch)
			}
			end()

			if len(budgets) == 0 && ch.Printer.Format() == printer.Human {
				if flags.page > 0 {
					ch.Printer.Println("No traffic budgets found on this page.")
				} else {
					ch.Printer.Printf("No traffic budgets affect fingerprint %s on %s/%s.\n",
						printer.BoldBlue(fingerprint), printer.BoldBlue(database), printer.BoldBlue(branch))
				}
				return nil
			}

			return ch.Printer.PrintResource(trafficcontrol.ToTrafficBudgets(budgets))
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace for the fingerprint")
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 25, "Number of results per page")

	return cmd
}
