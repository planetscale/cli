package maintenance

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ShowCmd shows a single maintenance schedule.
func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <schedule-id>",
		Short: "Show a maintenance schedule",
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching maintenance schedule %s for %s",
				printer.BoldBlue(scheduleID), printer.BoldBlue(database)))
			defer end()

			schedule, err := client.MaintenanceSchedules.Get(ctx, &ps.GetMaintenanceScheduleRequest{
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

			return ch.Printer.PrintResource(toMaintenanceSchedule(schedule))
		},
	}

	return cmd
}
