package branch

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

var maintenanceEventStates = []string{"scheduled", "in_progress", "completed", "canceled", "expired"}

func MaintenanceEventCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event <command>",
		Short: "List, show, and reschedule maintenance events for a Postgres branch",
		Long: `List, show, and reschedule maintenance events for a Postgres branch.
Postgres only.

A maintenance event is an update PlanetScale has scheduled for a branch, such
as an image update that restarts it. It runs in the branch's maintenance window
and must run by its deadline (not_after).`,
	}

	cmd.AddCommand(MaintenanceEventListCmd(ch))
	cmd.AddCommand(MaintenanceEventShowCmd(ch))
	cmd.AddCommand(MaintenanceEventRescheduleCmd(ch))

	return cmd
}

func MaintenanceEventListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		state   string
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:   "list <database> <branch>",
		Short: "List maintenance events for a Postgres branch",
		Long: `List maintenance events for a Postgres branch, latest scheduled time first. Use
--state to only list events in one state.`,
		Example: `  pscale branch maintenance event list mydb main --state scheduled`,
		Aliases: []string{"ls"},
		Args:    cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			// The API answers an unknown state with an empty list.
			if flags.state != "" && !slices.Contains(maintenanceEventStates, flags.state) {
				return fmt.Errorf("invalid --state %q: use one of %s", flags.state, strings.Join(maintenanceEventStates, ", "))
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "The 'branch maintenance event' commands"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching maintenance events for branch %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			events, err := client.MaintenanceEvents.List(ctx, &ps.ListMaintenanceEventsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				State:        flags.state,
			}, ps.WithPage(flags.page), ps.WithPerPage(flags.perPage))
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err, "read_database",
						"maintenance events are not available for database %s", printer.BoldBlue(database))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(events) == 0 && ch.Printer.Format() == printer.Human {
				if flags.page > 0 {
					ch.Printer.Println("No maintenance events found on this page.")
				} else {
					ch.Printer.Printf("No maintenance events exist for branch %s in %s.\n",
						printer.BoldBlue(branch), printer.BoldBlue(database))
				}
				return nil
			}

			return ch.Printer.PrintResource(toMaintenanceEvents(events))
		},
	}

	cmd.Flags().StringVar(&flags.state, "state", "",
		"Only list events in this state: "+strings.Join(maintenanceEventStates, ", "))
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")

	return cmd
}

func MaintenanceEventShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <branch> <id>",
		Short: "Show a maintenance event for a Postgres branch",
		Long: `Show a maintenance event for a Postgres branch: what it changes, why, the
impact to expect, when it is scheduled, its deadline, and the times it can be
rescheduled to.`,
		Args: cmdutil.RequiredArgs("database", "branch", "id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, id := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "The 'branch maintenance event' commands"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching maintenance event %s for branch %s in %s...",
				printer.BoldBlue(id), printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			event, err := getMaintenanceEvent(ctx, cmd, ch, client, database, branch, id)
			if err != nil {
				return err
			}
			end()

			if err := ch.Printer.PrintResource(toMaintenanceEvent(event)); err != nil {
				return err
			}

			if ch.Printer.Format() == printer.Human {
				if event.Why != "" {
					ch.Printer.Printf("Why: %s\n", event.Why)
				}
				if event.ExpectedImpact != "" {
					ch.Printer.Printf("Expected impact: %s\n", event.ExpectedImpact)
				}
				if len(event.AvailableTimes) > 0 {
					ch.Printer.Printf("Can be rescheduled to:\n%s\n", formatMaintenanceEventTimes(event.AvailableTimes))
				}
			}

			return nil
		},
	}

	return cmd
}

func MaintenanceEventRescheduleCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		to string
	}

	cmd := &cobra.Command{
		Use:   "reschedule <database> <branch> <id>",
		Short: "Reschedule a maintenance event for a Postgres branch",
		Long: `Move a maintenance event to another of its available times: the branch's
upcoming maintenance windows before the event's deadline. Run
'pscale branch maintenance event show' to list them.

An event can be rescheduled a limited number of times, and not once it has
started or while the maintenance window is open.`,
		Example: `  # Show the times the event can be moved to
  pscale branch maintenance event show mydb main x8k2m4p9q1rz

  # Move it to one of them
  pscale branch maintenance event reschedule mydb main x8k2m4p9q1rz --to 2026-11-08T03:00:00Z`,
		Args: cmdutil.RequiredArgs("database", "branch", "id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, id := args[0], args[1], args[2]

			to, err := time.Parse(time.RFC3339, flags.to)
			if err != nil {
				return fmt.Errorf("invalid --to %q: use an RFC 3339 time such as 2026-10-25T03:00:00Z", flags.to)
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "The 'branch maintenance event' commands"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Rescheduling maintenance event %s for branch %s in %s...",
				printer.BoldBlue(id), printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			event, err := getMaintenanceEvent(ctx, cmd, ch, client, database, branch, id)
			if err != nil {
				return err
			}

			if len(event.AvailableTimes) == 0 {
				return maintenanceEventNotReschedulableError(event)
			}
			if !slices.ContainsFunc(event.AvailableTimes, to.Equal) {
				return fmt.Errorf("maintenance event %s cannot be rescheduled to %s; it can be rescheduled to:\n%s",
					printer.BoldBlue(id), to.UTC().Format(time.RFC3339), formatMaintenanceEventTimes(event.AvailableTimes))
			}

			event, err = client.MaintenanceEvents.Reschedule(ctx, &ps.RescheduleMaintenanceEventRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				ID:           id,
				ScheduledAt:  to.UTC(),
			})
			if err != nil {
				switch {
				case maintenanceAPICode(err) == "forbidden":
					return maintenanceForbiddenError(ch.Config, fmt.Sprintf("reschedule maintenance for branch %s in database %s",
						printer.BoldBlue(branch), printer.BoldBlue(database)))
				case cmdutil.ErrCode(err) == ps.ErrNotFound:
					return maintenanceEventNotFoundError(database, branch, id)
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Maintenance event %s for branch %s in %s was rescheduled to %s.\n",
					printer.BoldBlue(id), printer.BoldBlue(branch), printer.BoldBlue(database),
					printer.BoldBlue(to.UTC().Format(maintenanceTimeFormat)))
				return nil
			}

			return ch.Printer.PrintResource(toMaintenanceEvent(event))
		},
	}

	cmd.Flags().StringVar(&flags.to, "to", "",
		"Time to move the event to, in RFC 3339 (e.g. 2026-11-08T03:00:00Z). Must be one of the event's available times")
	cmd.MarkFlagRequired("to") // nolint:errcheck

	return cmd
}

// getMaintenanceEvent fetches an event and checks that it belongs to the given
// branch. Events are addressed per database, so an ID from another branch
// would otherwise be accepted.
func getMaintenanceEvent(ctx context.Context, cmd *cobra.Command, ch *cmdutil.Helper, client *ps.Client, database, branch, id string) (*ps.MaintenanceEvent, error) {
	event, err := client.MaintenanceEvents.Get(ctx, &ps.GetMaintenanceEventRequest{
		Organization: ch.Config.Organization,
		Database:     database,
		ID:           id,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case ps.ErrNotFound:
			return nil, cmdutil.HandleNotFoundWithServiceTokenCheck(
				ctx, cmd, ch.Config, ch.Client, err, "read_database",
				"%s", maintenanceEventNotFoundError(database, branch, id))
		default:
			return nil, cmdutil.HandleError(err)
		}
	}

	if event.Branch == nil || event.Branch.Name != branch {
		return nil, maintenanceEventNotFoundError(database, branch, id)
	}

	return event, nil
}

func maintenanceEventNotFoundError(database, branch, id string) error {
	return fmt.Errorf("maintenance event %s does not exist for branch %s in database %s",
		printer.BoldBlue(id), printer.BoldBlue(branch), printer.BoldBlue(database))
}

// maintenanceEventNotReschedulableError explains why an event has no available
// times, checking the reasons in the order the API does.
func maintenanceEventNotReschedulableError(event *ps.MaintenanceEvent) error {
	id := printer.BoldBlue(event.ID)
	switch {
	case event.State != "scheduled":
		return fmt.Errorf("maintenance event %s is %s and cannot be rescheduled", id, event.State)
	case event.ReschedulesRemaining <= 0:
		return fmt.Errorf("maintenance event %s has no reschedules remaining", id)
	default:
		return fmt.Errorf("maintenance event %s cannot be rescheduled right now", id)
	}
}

func formatMaintenanceEventTimes(times []time.Time) string {
	lines := make([]string, 0, len(times))
	for _, t := range times {
		lines = append(lines, "  "+t.UTC().Format(time.RFC3339))
	}
	return strings.Join(lines, "\n")
}

// MaintenanceEvent is the human/JSON/CSV view of a maintenance event.
type MaintenanceEvent struct {
	ID                   string `header:"id" json:"id"`
	Branch               string `header:"branch" json:"branch"`
	State                string `header:"state" json:"state"`
	What                 string `header:"what" json:"what"`
	ScheduledAt          string `header:"scheduled_at" json:"scheduled_at"`
	NotAfter             string `header:"not_after" json:"not_after"`
	CompletedAt          string `header:"completed_at" json:"completed_at"`
	ReschedulesRemaining int    `header:"reschedules_remaining" json:"reschedules_remaining"`

	orig *ps.MaintenanceEvent
}

func (e *MaintenanceEvent) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(e.orig, "", "  ")
}

func (e *MaintenanceEvent) MarshalCSVValue() interface{} {
	return []*MaintenanceEvent{e}
}

func toMaintenanceEvent(event *ps.MaintenanceEvent) *MaintenanceEvent {
	out := &MaintenanceEvent{
		ID:                   event.ID,
		State:                event.State,
		What:                 event.What,
		ScheduledAt:          formatMaintenanceEventTime(event.ScheduledAt),
		NotAfter:             formatMaintenanceEventTime(&event.NotAfter),
		CompletedAt:          formatMaintenanceEventTime(event.CompletedAt),
		ReschedulesRemaining: event.ReschedulesRemaining,
		orig:                 event,
	}
	if event.Branch != nil {
		out.Branch = event.Branch.Name
	}
	return out
}

func toMaintenanceEvents(events []*ps.MaintenanceEvent) []*MaintenanceEvent {
	out := make([]*MaintenanceEvent, 0, len(events))
	for _, event := range events {
		out = append(out, toMaintenanceEvent(event))
	}
	return out
}

func formatMaintenanceEventTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.UTC().Format(maintenanceTimeFormat)
}
