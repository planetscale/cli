package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// AgentRunFeedbackCmd rates a completed Insights Agent run.
func AgentRunFeedbackCmd(ch *cmdutil.Helper) *cobra.Command {
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
	cmd.Flags().StringVar(&flags.reason, "reason", "", "Why the run was not useful: wrong_diagnosis, fix_wont_work, too_risky, already_known, doesnt_fit, or other")
	cmd.Flags().StringVar(&flags.note, "note", "", "A note about the run, up to 2000 characters")
	cmd.MarkFlagRequired("rating") // nolint:errcheck

	return cmd
}
