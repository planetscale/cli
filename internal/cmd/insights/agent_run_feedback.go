package insights

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// feedbackReasons matches the reason labels on the run page in the dashboard.
var feedbackReasons = []struct{ value, label string }{
	{"wrong_diagnosis", "Wrong diagnosis"},
	{"fix_wont_work", "Fix wouldn’t work"},
	{"too_risky", "Too risky"},
	{"already_known", "Already knew this"},
	{"doesnt_fit", "Doesn’t fit our code or setup"},
	{"other", "Other"},
}

// AgentRunFeedbackCmd rates a completed Insights Agent run.
func AgentRunFeedbackCmd(ch *cmdutil.Helper) *cobra.Command {
	reasonValues := make([]string, 0, len(feedbackReasons))
	reasonHelp := make([]string, 0, len(feedbackReasons))
	reasonCompletions := make([]cobra.Completion, 0, len(feedbackReasons))
	for _, r := range feedbackReasons {
		reasonValues = append(reasonValues, r.value)
		reasonHelp = append(reasonHelp, fmt.Sprintf("%s (%s)", r.value, r.label))
		reasonCompletions = append(reasonCompletions, cobra.CompletionWithDesc(r.value, r.label))
	}

	var flags struct {
		rating string
		reason string
		note   string
	}

	cmd := &cobra.Command{
		Use:   "feedback <database> <branch> <number>",
		Short: "Rate a completed Insights Agent run",
		Long: `Rate a completed Insights Agent run as useful or not useful. Rating a run
again replaces your earlier feedback.`,
		Example: `  pscale insights agent runs feedback mydb main 12 --org myorg --rating useful
  pscale insights agent runs feedback mydb main 12 --org myorg --rating not_useful --reason too_risky --note "Drops an index we still need"`,
		Args: cmdutil.RequiredArgs("database", "branch", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]
			number, err := parseRunNumber(args[2])
			if err != nil {
				return err
			}
			if flags.rating != "useful" && flags.rating != "not_useful" {
				return fmt.Errorf("invalid --rating %q: must be useful or not_useful", flags.rating)
			}
			if flags.reason != "" && !slices.Contains(reasonValues, flags.reason) {
				return fmt.Errorf("invalid --reason %q: must be one of %s", flags.reason, strings.Join(reasonValues, ", "))
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Rating Insights Agent run #%d on %s/%s...",
				number, printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			feedback, err := client.InsightsAgent.RateRun(ctx, &ps.RateInsightsAgentRunRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Number:       number,
				Rating:       flags.rating,
				Reason:       flags.reason,
				Note:         flags.note,
			})
			if err != nil {
				return agentRunNotFoundError(ch, err, database, branch, number)
			}
			end()

			switch ch.Printer.Format() {
			case printer.JSON:
				return ch.Printer.PrintJSON(feedback)
			case printer.Human:
				ch.Printer.Printf("Rated Insights Agent run %s on %s/%s as %s.\n",
					printer.BoldBlue(fmt.Sprintf("#%d", number)), printer.BoldBlue(database), printer.BoldBlue(branch), printer.Bold(feedback.Rating))
				return nil
			default:
				return ch.Printer.PrintResource([]*ps.InsightsAgentRunFeedback{feedback})
			}
		},
	}

	cmd.Flags().StringVar(&flags.rating, "rating", "", "Whether the run was useful: useful or not_useful")
	cmd.Flags().StringVar(&flags.reason, "reason", "", "Why the run was not useful: "+strings.Join(reasonHelp, ", "))
	cmd.Flags().StringVar(&flags.note, "note", "", "A note about the run, up to 2000 characters")
	cmd.MarkFlagRequired("rating") // nolint:errcheck

	cmd.RegisterFlagCompletionFunc("rating", cobra.FixedCompletions([]cobra.Completion{ // nolint:errcheck
		cobra.CompletionWithDesc("useful", "Useful"),
		cobra.CompletionWithDesc("not_useful", "Not useful"),
	}, cobra.ShellCompDirectiveNoFileComp))
	cmd.RegisterFlagCompletionFunc("reason", cobra.FixedCompletions(reasonCompletions, cobra.ShellCompDirectiveNoFileComp)) // nolint:errcheck

	return cmd
}
