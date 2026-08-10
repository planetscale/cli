package backup

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// PolicyUpdateCmd updates a backup policy.
func PolicyUpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name           string
		target         string
		retentionValue int
		retentionUnit  string
		frequencyValue int
		frequencyUnit  string
		scheduleTime   string
		scheduleDay    int
		scheduleWeek   int
	}

	cmd := &cobra.Command{
		Use:   "update <database> <policy-id>",
		Short: "Update a backup policy",
		Long: `Update a backup policy.

Only flags you pass are sent to the API. Required system policies may reject
some changes.`,
		Args: cmdutil.RequiredArgs("database", "policy-id"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			policyID := args[1]

			req := &ps.UpdateBackupPolicyRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Policy:       policyID,
			}

			changed := false
			if cmd.Flags().Changed("name") {
				req.Name = &flags.name
				changed = true
			}
			if cmd.Flags().Changed("target") {
				req.Target = &flags.target
				changed = true
			}
			if cmd.Flags().Changed("retention-value") {
				req.RetentionValue = &flags.retentionValue
				changed = true
			}
			if cmd.Flags().Changed("retention-unit") {
				req.RetentionUnit = &flags.retentionUnit
				changed = true
			}
			if cmd.Flags().Changed("frequency-value") {
				req.FrequencyValue = &flags.frequencyValue
				changed = true
			}
			if cmd.Flags().Changed("frequency-unit") {
				req.FrequencyUnit = &flags.frequencyUnit
				changed = true
			}
			if cmd.Flags().Changed("schedule-time") {
				req.ScheduleTime = &flags.scheduleTime
				changed = true
			}
			if cmd.Flags().Changed("schedule-day") {
				req.ScheduleDay = &flags.scheduleDay
				changed = true
			}
			if cmd.Flags().Changed("schedule-week") {
				req.ScheduleWeek = &flags.scheduleWeek
				changed = true
			}

			if !changed {
				return fmt.Errorf("at least one policy flag must be provided")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating backup policy %s for %s", printer.BoldBlue(policyID), printer.BoldBlue(database)))
			defer end()

			policy, err := client.BackupPolicies.Update(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("backup policy %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(policyID), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toBackupPolicy(policy))
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name for the backup policy")
	cmd.Flags().StringVar(&flags.target, "target", "", "Branch target: production or development")
	cmd.Flags().IntVar(&flags.retentionValue, "retention-value", 0, "Retention period value")
	cmd.Flags().StringVar(&flags.retentionUnit, "retention-unit", "", "Retention unit: hour, day, week, month, or year")
	cmd.Flags().IntVar(&flags.frequencyValue, "frequency-value", 0, "Frequency value")
	cmd.Flags().StringVar(&flags.frequencyUnit, "frequency-unit", "", "Frequency unit: hour, day, week, or month")
	cmd.Flags().StringVar(&flags.scheduleTime, "schedule-time", "", "Schedule time of day in HH:MM format")
	cmd.Flags().IntVar(&flags.scheduleDay, "schedule-day", 0, "Day of week (0=Sunday … 6=Saturday)")
	cmd.Flags().IntVar(&flags.scheduleWeek, "schedule-week", 0, "Week of month (0=first … 3=fourth)")

	return cmd
}
