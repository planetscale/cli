package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// AgentScheduleUpdateCmd creates or changes a branch's schedule for one run type.
func AgentScheduleUpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		enabled      bool
		frequency    string
		scheduleTime string
		timeZone     string
		dayOfWeek    int
		dayOfMonth   int
	}

	cmd := &cobra.Command{
		Use:   "update <database> <branch> <run-type>",
		Short: "Turn on, turn off, or change the schedule for a run type",
		Long: `Turn on, turn off, or change when the Insights Agent runs on its own for one
run type on a branch. <run-type> is slow_query or schema_recommendation.

Scheduled runs choose their own targets: slow query runs look at the queries
Insights recorded in the last 24 hours, and schema recommendation runs work on
the branch's open recommendations. A run with nothing to work on is skipped.

Flags that are not passed keep their current value.`,
		Example: `  pscale insights agent schedules update mydb main slow_query --org myorg --enabled --frequency week --day-of-week 1 --time 03:00 --time-zone America/New_York
  pscale insights agent schedules update mydb main schema_recommendation --org myorg --enabled=false`,
		Args: cmdutil.RequiredArgs("database", "branch", "run-type"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, runType := args[0], args[1], args[2]

			request := &ps.UpdateInsightsAgentScheduleRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				RunType:      runType,
			}
			changed := cmd.Flags().Changed
			if changed("enabled") {
				request.Enabled = &flags.enabled
			}
			if changed("frequency") {
				request.FrequencyUnit = &flags.frequency
			}
			if changed("time") {
				request.ScheduleTime = &flags.scheduleTime
			}
			if changed("time-zone") {
				request.TimeZone = &flags.timeZone
			}
			if changed("day-of-week") {
				request.DayOfWeek = &flags.dayOfWeek
			}
			if changed("day-of-month") {
				request.DayOfMonth = &flags.dayOfMonth
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating the %s schedule on %s/%s...",
				printer.BoldBlue(runType), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			schedule, err := client.InsightsAgent.UpdateSchedule(ctx, request)
			if err != nil {
				return agentNotFoundError(ch, err, database, branch)
			}
			end()

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(schedule)
			}
			return ch.Printer.PrintResource(agentScheduleRow(schedule))
		},
	}

	cmd.Flags().BoolVar(&flags.enabled, "enabled", false, "Whether the schedule runs")
	cmd.Flags().StringVar(&flags.frequency, "frequency", "", "How often the schedule runs: day, week, or month")
	cmd.Flags().StringVar(&flags.scheduleTime, "time", "", "The local time of day to run, in HH:MM format")
	cmd.Flags().StringVar(&flags.timeZone, "time-zone", "", "The IANA time zone for --time, such as America/New_York")
	cmd.Flags().IntVar(&flags.dayOfWeek, "day-of-week", 0, "The day of the week for weekly schedules, from 0 (Sunday) to 6 (Saturday)")
	cmd.Flags().IntVar(&flags.dayOfMonth, "day-of-month", 0, "The day of the month for monthly schedules, from 1 to 31. Shorter months run on their last day.")
	cmd.MarkFlagsOneRequired("enabled", "frequency", "time", "time-zone", "day-of-week", "day-of-month")

	return cmd
}
