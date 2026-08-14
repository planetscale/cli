package maintenance

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ListCmd lists maintenance schedules for a database.
func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database>",
		Short:   "List maintenance schedules for a database",
		Args:    cmdutil.RequiredArgs("database"),
		Aliases: []string{"ls"},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching maintenance schedules for %s", printer.BoldBlue(database)))
			defer end()

			schedules, err := client.MaintenanceSchedules.List(ctx, &ps.ListMaintenanceSchedulesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(schedules) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No maintenance schedules exist for %s.\n", printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(toMaintenanceSchedules(schedules))
		},
	}

	return cmd
}
