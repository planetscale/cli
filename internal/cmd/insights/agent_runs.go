package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// AgentRunRow is an Insights Agent run formatted for table output.
type AgentRunRow struct {
	Number    int    `header:"number" json:"number"`
	Type      string `header:"type" json:"run_type"`
	State     string `header:"state" json:"state"`
	Outcome   string `header:"outcome" json:"outcome"`
	Title     string `header:"title" json:"title"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
}

// AgentRunsCmd lists a branch's Insights Agent runs.
func AgentRunsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		fingerprint          string
		keyspace             string
		schemaRecommendation int
		page                 int
		perPage              int
	}

	cmd := &cobra.Command{
		Use:     "runs <database> <branch>",
		Short:   "List Insights Agent runs on a branch, newest first",
		Aliases: []string{"run"},
		Example: `  pscale insights agent runs mydb main --org myorg
  pscale insights agent runs mydb main --org myorg --fingerprint 59801dae501c --keyspace mydb
  pscale insights agent runs mydb main --org myorg --schema-recommendation 3`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			if flags.keyspace != "" && flags.fingerprint == "" {
				return fmt.Errorf("--keyspace requires --fingerprint")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching Insights Agent runs for %s/%s...",
				printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			runs, err := client.InsightsAgent.ListRuns(ctx, &ps.ListInsightsAgentRunsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			}, ps.WithFingerprint(flags.fingerprint), ps.WithKeyspace(flags.keyspace),
				ps.WithSchemaRecommendation(flags.schemaRecommendation), ps.WithPage(flags.page), ps.WithPerPage(flags.perPage))
			if err != nil {
				return agentNotFoundError(ch, err, database, branch)
			}
			end()

			if len(runs) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No Insights Agent runs found on %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(runs)
			}

			rows := make([]*AgentRunRow, 0, len(runs))
			for _, run := range runs {
				rows = append(rows, agentRunRow(run))
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	cmd.Flags().StringVar(&flags.fingerprint, "fingerprint", "", "Only list runs that reviewed the slow query with this fingerprint")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace of the --fingerprint query")
	cmd.Flags().IntVar(&flags.schemaRecommendation, "schema-recommendation", 0, "Only list runs that implemented the schema recommendation with this number")
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 25, "Number of results per page")

	cmd.AddCommand(AgentRunShowCmd(ch))
	cmd.AddCommand(AgentRunCreateCmd(ch))
	cmd.AddCommand(AgentRunCancelCmd(ch))
	cmd.AddCommand(AgentRunFeedbackCmd(ch))

	return cmd
}

func agentRunRow(run *ps.InsightsAgentRun) *AgentRunRow {
	return &AgentRunRow{
		Number:    run.Number,
		Type:      run.RunType,
		State:     run.State,
		Outcome:   run.Outcome,
		Title:     truncate(run.Title, 80),
		CreatedAt: printer.GetMilliseconds(run.CreatedAt),
	}
}
