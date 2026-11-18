package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// AgentRunShowCmd shows one Insights Agent run with its targets and stages.
func AgentRunShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <branch> <number>",
		Short: "Show an Insights Agent run, its stages, and the pull request it opened",
		Example: `  pscale insights agent runs show mydb main 12 --org myorg
  pscale insights agent runs show mydb main 12 --org myorg --format json`,
		Args: cmdutil.RequiredArgs("database", "branch", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]
			number, err := parseRunNumber(args[2])
			if err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching Insights Agent run #%d for %s/%s...",
				number, printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			run, err := client.InsightsAgent.GetRun(ctx, &ps.GetInsightsAgentRunRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Number:       number,
			})
			if err != nil {
				return agentRunNotFoundError(ch, err, database, branch, number)
			}
			end()

			switch ch.Printer.Format() {
			case printer.JSON:
				return ch.Printer.PrintJSON(run)
			case printer.Human:
				printAgentRun(ch, run)
				return nil
			default:
				return ch.Printer.PrintResource([]*AgentRunRow{agentRunRow(run)})
			}
		},
	}

	return cmd
}

func printAgentRun(ch *cmdutil.Helper, run *ps.InsightsAgentRun) {
	field := func(label, value string) {
		if value != "" {
			ch.Printer.Printf("%-14s%s\n", label+":", value)
		}
	}

	field("Number", fmt.Sprintf("%d", run.Number))
	field("Type", run.RunType)
	field("State", run.State)
	field("Outcome", run.Outcome)
	field("Title", run.Title)
	if run.Actor != nil {
		field("Started by", run.Actor.Name)
	}
	if run.Repository != nil {
		field("Repository", fmt.Sprintf("%s/%s@%s", run.Repository.Owner, run.Repository.Name, shortCommit(run.Repository.Commit)))
	}
	if run.PullRequest != nil {
		field("Pull request", fmt.Sprintf("%s (%s)", run.PullRequest.URL, run.PullRequest.State))
		if run.PullRequest.DeployRequest != nil {
			field("Deploy request", run.PullRequest.DeployRequest.HtmlURL)
		}
	}
	field("Failure", run.FailureMessage)
	field("URL", run.HtmlURL)

	if len(run.QueryPatterns) > 0 {
		ch.Printer.Printf("\n%s\n", printer.Bold("Query patterns:"))
		for _, pattern := range run.QueryPatterns {
			ch.Printer.Printf("  %s (%s)\n", pattern.Fingerprint, pattern.Keyspace)
		}
	}
	if len(run.SchemaRecommendations) > 0 {
		ch.Printer.Printf("\n%s\n", printer.Bold("Schema recommendations:"))
		for _, recommendation := range run.SchemaRecommendations {
			ch.Printer.Printf("  #%d %s (%s)\n", recommendation.Number, recommendation.Title, recommendation.State)
		}
	}
	if len(run.Stages) > 0 {
		ch.Printer.Printf("\n%s\n", printer.Bold("Stages:"))
		for _, stage := range run.Stages {
			line := fmt.Sprintf("  %-24s%s", stage.Name, stage.Status)
			if stage.Message != "" {
				line += ": " + stage.Message
			}
			ch.Printer.Println(line)
		}
	}
	if run.Summary != "" {
		ch.Printer.Printf("\n%s\n%s\n", printer.Bold("Summary:"), run.Summary)
	}
}

func shortCommit(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}
