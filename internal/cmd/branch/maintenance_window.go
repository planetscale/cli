package branch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

const (
	// maintenanceSchedulesPerPage is the largest page the API serves.
	maintenanceSchedulesPerPage = 100

	maintenanceTimeFormat = "2006-01-02 15:04 MST"
)

func MaintenanceWindowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "window <command>",
		Short: "Show and set the maintenance window of a Postgres branch",
		Long: `Show and set the weekly maintenance window of a Postgres branch. Postgres only.

The maintenance window is the day and hour, in UTC, when scheduled maintenance
for the branch is allowed to start.`,
	}

	cmd.AddCommand(MaintenanceWindowShowCmd(ch))
	cmd.AddCommand(MaintenanceWindowSetCmd(ch))

	return cmd
}

func MaintenanceWindowShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <branch>",
		Short: "Show the maintenance window of a Postgres branch",
		Long: `Show the weekly maintenance window of a Postgres branch: its day, its hour in
UTC, how long it lasts, and when it next opens.`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "The 'branch maintenance window' commands"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching maintenance window for branch %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			schedule, err := findMaintenanceWindow(ctx, cmd, ch, client, database, branch)
			if err != nil {
				return err
			}
			end()

			return ch.Printer.PrintResource(toMaintenanceWindow(schedule))
		},
	}

	return cmd
}

func MaintenanceWindowSetCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		day  string
		hour int
	}

	cmd := &cobra.Command{
		Use:   "set <database> <branch>",
		Short: "Change the maintenance window of a Postgres branch",
		Long: `Set the day and hour, in UTC, when the weekly maintenance window of a Postgres
branch opens. At least one of --day and --hour is required; the one you leave
out keeps its current value.

Scheduled maintenance moves to the new window. Check it afterwards with
'pscale branch maintenance event list'.

You can't change the window while it is open, or if the change would push
scheduled maintenance past its deadline.`,
		Example: `  pscale branch maintenance window set mydb main --day sunday --hour 3
  pscale branch maintenance window set mydb main --hour 14`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			req := &ps.UpdateMaintenanceScheduleRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			}
			if cmd.Flags().Changed("day") {
				day, err := parseMaintenanceDay(flags.day)
				if err != nil {
					return err
				}
				req.Day = &day
			}
			if cmd.Flags().Changed("hour") {
				if flags.hour < 0 || flags.hour > 23 {
					return fmt.Errorf("invalid --hour %d: use an hour from 0 to 23", flags.hour)
				}
				req.Hour = &flags.hour
			}
			if req.Day == nil && req.Hour == nil {
				return errors.New("at least one of --day or --hour is required")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "The 'branch maintenance window' commands"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Setting maintenance window for branch %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			schedule, err := findMaintenanceWindow(ctx, cmd, ch, client, database, branch)
			if err != nil {
				return err
			}

			// The API records an audit entry for every update, so do not send
			// one that changes nothing.
			if (req.Day == nil || *req.Day == schedule.Day) && (req.Hour == nil || *req.Hour == schedule.Hour) {
				end()
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Maintenance window for branch %s in %s is unchanged.\n",
						printer.BoldBlue(branch), printer.BoldBlue(database))
					return nil
				}
				return ch.Printer.PrintResource(toMaintenanceWindow(schedule))
			}

			req.Schedule = schedule.ID
			schedule, err = client.MaintenanceSchedules.Update(ctx, req)
			if err != nil {
				switch {
				case maintenanceAPICode(err) == "forbidden":
					return maintenanceForbiddenError(ch.Config, fmt.Sprintf("change the maintenance window of branch %s in database %s",
						printer.BoldBlue(branch), printer.BoldBlue(database)))
				case cmdutil.ErrCode(err) == ps.ErrNotFound:
					return fmt.Errorf("the maintenance window of branch %s in database %s could not be found; it may have been replaced since it was read, so run the command again",
						printer.BoldBlue(branch), printer.BoldBlue(database))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toMaintenanceWindow(schedule))
		},
	}

	cmd.Flags().StringVar(&flags.day, "day", "",
		"Day of the week the window opens, in UTC: a day name or a number from 0 (Sunday) to 6 (Saturday)")
	cmd.Flags().IntVar(&flags.hour, "hour", 0, "Hour of the day the window opens, in UTC (0-23)")

	return cmd
}

// findMaintenanceWindow returns the weekly maintenance schedule of a branch.
// Schedules are only listed per database, so it pages through them. A branch
// can also hold one-time schedules, and a deleted branch can share a name with
// a live one; neither is the branch's window.
func findMaintenanceWindow(ctx context.Context, cmd *cobra.Command, ch *cmdutil.Helper, client *ps.Client, database, branch string) (*ps.MaintenanceSchedule, error) {
	for page := 1; ; page++ {
		schedules, err := client.MaintenanceSchedules.List(ctx, &ps.ListMaintenanceSchedulesRequest{
			Organization: ch.Config.Organization,
			Database:     database,
		}, ps.WithPage(page), ps.WithPerPage(maintenanceSchedulesPerPage))
		if err != nil {
			switch {
			case maintenanceAPICode(err) == "method_not_allowed":
				return nil, fmt.Errorf("maintenance windows are not available for database %s", printer.BoldBlue(database))
			case cmdutil.ErrCode(err) == ps.ErrNotFound:
				return nil, cmdutil.HandleNotFoundWithServiceTokenCheck(
					ctx, cmd, ch.Config, ch.Client, err, "read_database",
					"database %s does not exist in organization %s",
					printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
			default:
				return nil, cmdutil.HandleError(err)
			}
		}

		for _, schedule := range schedules {
			b := schedule.DatabaseBranch
			if schedule.FrequencyUnit == "week" && b != nil && b.Name == branch && b.DeletedAt == nil {
				return schedule, nil
			}
		}

		// A short page is not taken as the last one: the API may serve fewer
		// than were asked for.
		if len(schedules) == 0 {
			return nil, fmt.Errorf("branch %s in database %s has no maintenance window",
				printer.BoldBlue(branch), printer.BoldBlue(database))
		}
	}
}

// maintenanceAPICode returns the API's own error code. The client only maps a
// few of them to an ErrorCode, and "forbidden" and "method_not_allowed" are
// not among those.
func maintenanceAPICode(err error) string {
	var perr *ps.Error
	if !errors.As(err, &perr) {
		return ""
	}
	return perr.APICode
}

func maintenanceForbiddenError(cfg *config.Config, action string) error {
	msg := fmt.Sprintf("you do not have permission to %s", action)
	if cfg.ServiceTokenIsSet() {
		msg += "\n\nNote: You are using a service token for authentication. It needs the 'write_database' permission on this database"
	}
	return errors.New(msg)
}

// parseMaintenanceDay accepts a day name or a number from 0 (Sunday) to 6
// (Saturday). The API also knows day 7, "every day", but refuses it for a
// weekly window.
func parseMaintenanceDay(value string) (int, error) {
	value = strings.TrimSpace(value)
	for day := time.Sunday; day <= time.Saturday; day++ {
		if strings.EqualFold(value, day.String()) {
			return int(day), nil
		}
	}
	if day, err := strconv.Atoi(value); err == nil && day >= 0 && day <= 6 {
		return day, nil
	}
	return 0, fmt.Errorf("invalid --day %q: use a day name (sunday to saturday) or a number from 0 (Sunday) to 6 (Saturday)", value)
}

// MaintenanceWindow is the human/JSON/CSV view of a branch's weekly
// maintenance window.
type MaintenanceWindow struct {
	Branch          string `header:"branch" json:"branch"`
	Day             string `header:"day" json:"day"`
	HourUTC         int    `header:"hour_utc" json:"hour_utc"`
	DurationHours   int    `header:"duration_hours" json:"duration_hours"`
	NextWindowUTC   string `header:"next_window_utc" json:"next_window_utc"`
	NextWindowLocal string `header:"next_window_local" json:"next_window_local"`

	orig *ps.MaintenanceSchedule
}

func (w *MaintenanceWindow) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(w.orig, "", "  ")
}

func (w *MaintenanceWindow) MarshalCSVValue() interface{} {
	return []*MaintenanceWindow{w}
}

func toMaintenanceWindow(schedule *ps.MaintenanceSchedule) *MaintenanceWindow {
	out := &MaintenanceWindow{
		Day:             time.Weekday(schedule.Day).String(),
		HourUTC:         schedule.Hour,
		DurationHours:   schedule.Duration,
		NextWindowUTC:   schedule.NextWindowDatetime.UTC().Format(maintenanceTimeFormat),
		NextWindowLocal: schedule.NextWindowDatetime.Local().Format(maintenanceTimeFormat),
		orig:            schedule,
	}
	if schedule.DatabaseBranch != nil {
		out.Branch = schedule.DatabaseBranch.Name
	}
	return out
}
