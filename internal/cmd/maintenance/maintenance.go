package maintenance

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// MaintenanceCmd manages maintenance schedules for a database.
func MaintenanceCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "maintenance <command>",
		Short: "List and show maintenance schedules for a database",
		Long: `List and show maintenance schedules for a Vitess database.

Maintenance schedules define when PlanetScale can perform planned maintenance
(for example version updates). Available for Vitess databases on Enterprise plans.`,
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(WindowsCmd(ch))

	return cmd
}

// MaintenanceSchedule is the human/JSON/CSV view of a maintenance schedule.
type MaintenanceSchedule struct {
	ID                   string `header:"id" json:"id"`
	Name                 string `header:"name" json:"name"`
	Enabled              bool   `header:"enabled" json:"enabled"`
	Required             bool   `header:"required" json:"required"`
	Frequency            string `header:"frequency" json:"frequency"`
	Day                  string `header:"day" json:"day"`
	Week                 string `header:"week" json:"week"`
	HourUTC              int    `header:"hour_utc" json:"hour_utc"`
	DurationHours        int    `header:"duration_hours" json:"duration_hours"`
	NextWindow           int64  `header:"next_window,timestamp(ms|utc|human)" json:"next_window"`
	LastWindow           int64  `header:"last_window,timestamp(ms|utc|human)" json:"last_window"`
	PendingVitessVersion string `header:"pending_vitess" json:"pending_vitess"`
	PendingMySQLVersion  string `header:"pending_mysql" json:"pending_mysql"`
	ExpiresAt            *int64 `header:"expires_at,timestamp(ms|utc|human)" json:"expires_at"`
	DeadlineAt           *int64 `header:"deadline_at,timestamp(ms|utc|human)" json:"deadline_at"`
	CreatedAt            int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt            int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.MaintenanceSchedule
}

func (s *MaintenanceSchedule) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(s.orig, "", "  ")
}

func (s *MaintenanceSchedule) MarshalCSVValue() interface{} {
	return []*MaintenanceSchedule{s}
}

func toMaintenanceSchedule(schedule *ps.MaintenanceSchedule) *MaintenanceSchedule {
	out := &MaintenanceSchedule{
		ID:            schedule.ID,
		Name:          schedule.Name,
		Enabled:       schedule.Enabled,
		Required:      schedule.Required,
		Frequency:     formatFrequency(schedule.FrequencyValue, schedule.FrequencyUnit),
		Day:           formatDay(schedule.Day),
		Week:          formatWeek(schedule.Week, schedule.FrequencyUnit),
		HourUTC:       schedule.Hour,
		DurationHours: schedule.Duration,
		NextWindow:    printer.GetMilliseconds(schedule.NextWindowDatetime),
		LastWindow:    printer.GetMilliseconds(schedule.LastWindowDatetime),
		ExpiresAt:     printer.GetMillisecondsIfExists(schedule.ExpiresAt),
		DeadlineAt:    printer.GetMillisecondsIfExists(schedule.DeadlineAt),
		CreatedAt:     printer.GetMilliseconds(schedule.CreatedAt),
		UpdatedAt:     printer.GetMilliseconds(schedule.UpdatedAt),
		orig:          schedule,
	}
	if schedule.PendingVitessVersion != nil && *schedule.PendingVitessVersion != "" {
		out.PendingVitessVersion = *schedule.PendingVitessVersion
	} else {
		out.PendingVitessVersion = "-"
	}
	if schedule.PendingMySQLVersion != nil && *schedule.PendingMySQLVersion != "" {
		out.PendingMySQLVersion = *schedule.PendingMySQLVersion
	} else {
		out.PendingMySQLVersion = "-"
	}
	return out
}

func toMaintenanceSchedules(schedules []*ps.MaintenanceSchedule) []*MaintenanceSchedule {
	out := make([]*MaintenanceSchedule, 0, len(schedules))
	for _, schedule := range schedules {
		out = append(out, toMaintenanceSchedule(schedule))
	}
	return out
}

func formatFrequency(value int, unit string) string {
	if unit == "" {
		return fmt.Sprintf("%d", value)
	}
	if value == 1 {
		return unit
	}
	return fmt.Sprintf("%d %ss", value, unit)
}

func formatDay(day int) string {
	switch day {
	case 0:
		return "Sunday"
	case 1:
		return "Monday"
	case 2:
		return "Tuesday"
	case 3:
		return "Wednesday"
	case 4:
		return "Thursday"
	case 5:
		return "Friday"
	case 6:
		return "Saturday"
	case 7:
		return "every day"
	default:
		return fmt.Sprintf("%d", day)
	}
}

func formatWeek(week int, frequencyUnit string) string {
	if frequencyUnit != "month" {
		return "-"
	}
	switch week {
	case 0:
		return "first"
	case 1:
		return "second"
	case 2:
		return "third"
	case 3:
		return "fourth"
	default:
		return fmt.Sprintf("%d", week)
	}
}
