package backup

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// PolicyCreateCmd creates a backup policy.
func PolicyCreateCmd(ch *cmdutil.Helper) *cobra.Command {
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
		Use:   "create <database>",
		Short: "Create a backup policy",
		Long: `Create a scheduled backup policy for production or development branches.

Custom schedules beyond the included defaults may incur additional backup storage charges.`,
		Args: cmdutil.RequiredArgs("database"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			req := &ps.CreateBackupPolicyRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Name:           flags.name,
				Target:         flags.target,
				RetentionValue: flags.retentionValue,
				RetentionUnit:  flags.retentionUnit,
				FrequencyValue: flags.frequencyValue,
				FrequencyUnit:  flags.frequencyUnit,
				ScheduleTime:   flags.scheduleTime,
			}
			if cmd.Flags().Changed("schedule-day") {
				req.ScheduleDay = &flags.scheduleDay
			}
			if cmd.Flags().Changed("schedule-week") {
				req.ScheduleWeek = &flags.scheduleWeek
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating backup policy for %s", printer.BoldBlue(database)))
			defer end()

			policy, err := client.BackupPolicies.Create(ctx, req)
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

			return ch.Printer.PrintResource(toBackupPolicy(policy))
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Optional name for the backup policy")
	cmd.Flags().StringVar(&flags.target, "target", "", "Branch target: production or development (required)")
	cmd.Flags().IntVar(&flags.retentionValue, "retention-value", 0, "Retention period value (required)")
	cmd.Flags().StringVar(&flags.retentionUnit, "retention-unit", "", "Retention unit: hour, day, week, month, or year (required)")
	cmd.Flags().IntVar(&flags.frequencyValue, "frequency-value", 0, "Frequency value (required)")
	cmd.Flags().StringVar(&flags.frequencyUnit, "frequency-unit", "", "Frequency unit: hour, day, week, or month (required)")
	cmd.Flags().StringVar(&flags.scheduleTime, "schedule-time", "", "Schedule time of day in HH:MM format (required)")
	cmd.Flags().IntVar(&flags.scheduleDay, "schedule-day", 0, "Day of week (0=Sunday … 6=Saturday); used for weekly/monthly schedules")
	cmd.Flags().IntVar(&flags.scheduleWeek, "schedule-week", 0, "Week of month (0=first … 3=fourth); used for monthly schedules")

	cmd.MarkFlagRequired("target")          // nolint:errcheck
	cmd.MarkFlagRequired("retention-value") // nolint:errcheck
	cmd.MarkFlagRequired("retention-unit")  // nolint:errcheck
	cmd.MarkFlagRequired("frequency-value") // nolint:errcheck
	cmd.MarkFlagRequired("frequency-unit")  // nolint:errcheck
	cmd.MarkFlagRequired("schedule-time")   // nolint:errcheck

	return cmd
}
