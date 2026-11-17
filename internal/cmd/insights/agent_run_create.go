package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// AgentRunCreateCmd starts an Insights Agent run on a branch.
func AgentRunCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		fingerprints          []string
		keyspace              string
		schemaRecommendations []int
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Start an Insights Agent run that reviews slow queries or implements schema recommendations",
		Long: `Start an Insights Agent run on a branch. The database needs a connected
repository, and the agent opens a pull request against it with the changes it
makes.

Pass up to five slow query fingerprints with --fingerprint and their --keyspace
to review slow queries, or up to five schema recommendation numbers with
--schema-recommendation to implement them. Find fingerprints with
"pscale insights queries" and recommendation numbers with
"pscale insights recommendations".`,
		Example: `  pscale insights agent runs create mydb main --org myorg --fingerprint 59801dae501c --keyspace mydb
  pscale insights agent runs create mydb main --org myorg --schema-recommendation 3,4`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			request := &ps.CreateInsightsAgentRunRequest{
				Organization:          ch.Config.Organization,
				Database:              database,
				Branch:                branch,
				SchemaRecommendations: flags.schemaRecommendations,
			}
			for _, fingerprint := range flags.fingerprints {
				request.QueryPatterns = append(request.QueryPatterns, &ps.InsightsAgentQueryPattern{
					Fingerprint: fingerprint,
					Keyspace:    flags.keyspace,
				})
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Starting an Insights Agent run on %s/%s...",
				printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			run, err := client.InsightsAgent.CreateRun(ctx, request)
			if err != nil {
				return agentNotFoundError(ch, err, database, branch)
			}
			end()

			switch ch.Printer.Format() {
			case printer.JSON:
				return ch.Printer.PrintJSON(run)
			case printer.Human:
				ch.Printer.Printf("Insights Agent run %s was started on %s/%s.\n",
					printer.BoldBlue(fmt.Sprintf("#%d", run.Number)), printer.BoldBlue(database), printer.BoldBlue(branch))
				ch.Printer.Printf("Follow it with: pscale insights agent runs show %s %s %d --org %s\n",
					database, branch, run.Number, ch.Config.Organization)
				return nil
			default:
				return ch.Printer.PrintResource([]*AgentRunRow{agentRunRow(run)})
			}
		},
	}

	cmd.Flags().StringSliceVar(&flags.fingerprints, "fingerprint", nil, "Fingerprint of a slow query to review (repeat or comma-separate)")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace of the --fingerprint queries")
	cmd.Flags().IntSliceVar(&flags.schemaRecommendations, "schema-recommendation", nil, "Number of a schema recommendation to implement (repeat or comma-separate)")
	cmd.MarkFlagsRequiredTogether("fingerprint", "keyspace")
	cmd.MarkFlagsOneRequired("fingerprint", "schema-recommendation")
	cmd.MarkFlagsMutuallyExclusive("fingerprint", "schema-recommendation")

	return cmd
}
