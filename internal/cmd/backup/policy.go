package backup

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// PolicyCmd manages database backup schedules/policies.
func PolicyCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy <command>",
		Short: "Create, list, show, update, and delete backup policies",
		Long: `Manage scheduled backup policies for a database.

Backup policies define automatic backup frequency, schedule, and retention for
production or development branches. This is separate from one-off branch
backups created with 'pscale backup create'.`,
	}

	cmd.AddCommand(PolicyListCmd(ch))
	cmd.AddCommand(PolicyShowCmd(ch))
	cmd.AddCommand(PolicyCreateCmd(ch))
	cmd.AddCommand(PolicyUpdateCmd(ch))
	cmd.AddCommand(PolicyDeleteCmd(ch))

	return cmd
}

// BackupPolicy is the human/JSON/CSV view of a backup policy.
type BackupPolicy struct {
	ID           string `header:"id" json:"id"`
	Name         string `header:"name" json:"name"`
	Target       string `header:"target" json:"target"`
	Retention    string `header:"retention" json:"retention"`
	Frequency    string `header:"frequency" json:"frequency"`
	ScheduleTime string `header:"schedule_time" json:"schedule_time"`
	ScheduleDay  string `header:"schedule_day" json:"schedule_day"`
	ScheduleWeek string `header:"schedule_week" json:"schedule_week"`
	Required     bool   `header:"required" json:"required"`
	LastRanAt    *int64 `header:"last_ran_at,timestamp(ms|utc|human)" json:"last_ran_at"`
	NextRunAt    *int64 `header:"next_run_at,timestamp(ms|utc|human)" json:"next_run_at"`
	CreatedAt    int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt    int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.BackupPolicy
}

func (p *BackupPolicy) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(p.orig, "", "  ")
}

func (p *BackupPolicy) MarshalCSVValue() interface{} {
	return []*BackupPolicy{p}
}

func toBackupPolicy(policy *ps.BackupPolicy) *BackupPolicy {
	out := &BackupPolicy{
		ID:           policy.ID,
		Name:         policy.Name,
		Target:       policy.Target,
		Retention:    formatQuantity(policy.RetentionValue, policy.RetentionUnit),
		Frequency:    formatQuantity(policy.FrequencyValue, policy.FrequencyUnit),
		ScheduleTime: policy.ScheduleTime,
		ScheduleDay:  formatOptionalInt(policy.ScheduleDay),
		ScheduleWeek: formatOptionalInt(policy.ScheduleWeek),
		Required:     policy.Required,
		LastRanAt:    printer.GetMillisecondsIfExists(policy.LastRanAt),
		NextRunAt:    printer.GetMillisecondsIfExists(policy.NextRunAt),
		CreatedAt:    printer.GetMilliseconds(policy.CreatedAt),
		UpdatedAt:    printer.GetMilliseconds(policy.UpdatedAt),
		orig:         policy,
	}
	return out
}

func toBackupPolicies(policies []*ps.BackupPolicy) []*BackupPolicy {
	out := make([]*BackupPolicy, 0, len(policies))
	for _, policy := range policies {
		out = append(out, toBackupPolicy(policy))
	}
	return out
}

func formatQuantity(value int, unit string) string {
	if unit == "" {
		return fmt.Sprintf("%d", value)
	}
	return fmt.Sprintf("%d %s", value, unit)
}

func formatOptionalInt(v *int) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *v)
}
