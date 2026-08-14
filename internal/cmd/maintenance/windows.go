package maintenance

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// WindowsCmd lists maintenance windows for a schedule.
func WindowsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "windows <database> <schedule-id>",
		Short: "List maintenance windows for a schedule",
		Args:  cmdutil.RequiredArgs("database", "schedule-id"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			scheduleID := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching maintenance windows for schedule %s on %s",
				printer.BoldBlue(scheduleID), printer.BoldBlue(database)))
			defer end()

			windows, err := client.MaintenanceSchedules.ListWindows(ctx, &ps.ListMaintenanceWindowsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Schedule:     scheduleID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("maintenance schedule %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(scheduleID), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(windows) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No maintenance windows exist for schedule %s on %s.\n",
					printer.BoldBlue(scheduleID), printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(toMaintenanceWindows(windows))
		},
	}

	return cmd
}

// MaintenanceWindow is the human/JSON/CSV view of a maintenance window.
type MaintenanceWindow struct {
	ID         string `header:"id" json:"id"`
	StartedAt  *int64 `header:"started_at,timestamp(ms|utc|human)" json:"started_at"`
	FinishedAt *int64 `header:"finished_at,timestamp(ms|utc|human)" json:"finished_at"`
	CreatedAt  int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt  int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.MaintenanceWindow
}

func (w *MaintenanceWindow) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(w.orig, "", "  ")
}

func (w *MaintenanceWindow) MarshalCSVValue() interface{} {
	return []*MaintenanceWindow{w}
}

func toMaintenanceWindow(window *ps.MaintenanceWindow) *MaintenanceWindow {
	return &MaintenanceWindow{
		ID:         window.ID,
		StartedAt:  printer.GetMillisecondsIfExists(window.StartedAt),
		FinishedAt: printer.GetMillisecondsIfExists(window.FinishedAt),
		CreatedAt:  printer.GetMilliseconds(window.CreatedAt),
		UpdatedAt:  printer.GetMilliseconds(window.UpdatedAt),
		orig:       window,
	}
}

func toMaintenanceWindows(windows []*ps.MaintenanceWindow) []*MaintenanceWindow {
	out := make([]*MaintenanceWindow, 0, len(windows))
	for _, window := range windows {
		out = append(out, toMaintenanceWindow(window))
	}
	return out
}
