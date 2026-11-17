package insights

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// AgentScheduleRow is an Insights Agent schedule formatted for table output.
type AgentScheduleRow struct {
	Type      string `header:"type" json:"run_type"`
	Enabled   bool   `header:"enabled" json:"enabled"`
	Frequency string `header:"frequency" json:"frequency_unit"`
	Day       string `header:"day" json:"day"`
	Time      string `header:"time" json:"schedule_time"`
	TimeZone  string `header:"time_zone" json:"time_zone"`
	NextRunAt *int64 `header:"next_run_at,timestamp(ms|utc|human)" json:"next_run_at"`
}

// AgentSchedulesCmd lists a branch's Insights Agent schedules.
func AgentSchedulesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedules <database> <branch>",
		Short:   "List the Insights Agent schedules on a branch",
		Aliases: []string{"schedule"},
		Example: `  pscale insights agent schedules mydb main --org myorg`,
		Args:    cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching Insights Agent schedules for %s/%s...",
				printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			schedules, err := client.InsightsAgent.ListSchedules(ctx, &ps.ListInsightsAgentSchedulesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return agentNotFoundError(ch, err, database, branch)
			}
			end()

			if len(schedules) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No Insights Agent schedules on %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(schedules)
			}

			rows := make([]*AgentScheduleRow, 0, len(schedules))
			for _, schedule := range schedules {
				rows = append(rows, agentScheduleRow(schedule))
			}
			return ch.Printer.PrintResource(rows)
		},
	}

	cmd.AddCommand(AgentScheduleUpdateCmd(ch))

	return cmd
}

func agentScheduleRow(schedule *ps.InsightsAgentSchedule) *AgentScheduleRow {
	row := &AgentScheduleRow{
		Type:      schedule.RunType,
		Enabled:   schedule.Enabled,
		Frequency: schedule.FrequencyUnit,
		Time:      schedule.ScheduleTime,
		TimeZone:  schedule.TimeZone,
		NextRunAt: printer.GetMillisecondsIfExists(schedule.NextRunAt),
	}
	switch {
	case schedule.FrequencyUnit == "week" && schedule.DayOfWeek != nil:
		row.Day = time.Weekday(*schedule.DayOfWeek).String()
	case schedule.FrequencyUnit == "month" && schedule.DayOfMonth != nil:
		row.Day = strconv.Itoa(*schedule.DayOfMonth)
	}
	return row
}
