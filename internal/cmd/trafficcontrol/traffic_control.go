package trafficcontrol

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func TrafficCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "traffic-control <command>",
		Short: "Manage Database Traffic Control™ for a Postgres database branch",
		Long: "Manage Database Traffic Control™ budgets and rules for a Postgres database branch.\n\n" +
			"This command is only supported for Postgres databases.",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	budgetCmd := &cobra.Command{
		Use:   "budget <command>",
		Short: "Manage traffic budgets",
	}
	budgetCmd.AddCommand(
		BudgetShowCmd(ch),
		BudgetCreateCmd(ch),
		BudgetUpdateCmd(ch),
	)

	ruleCmd := &cobra.Command{
		Use:   "rule <command>",
		Short: "Manage traffic rules",
	}
	ruleCmd.AddCommand(
		RuleCreateCmd(ch),
	)

	cmd.AddCommand(budgetCmd, ruleCmd)
	return cmd
}

type TrafficBudget struct {
	ID          string `header:"id" json:"id"`
	Name        string `header:"name" json:"name"`
	Mode        string `header:"mode" json:"mode"`
	Capacity    string `header:"capacity" json:"capacity"`
	Rate        string `header:"rate" json:"rate"`
	Burst       string `header:"burst" json:"burst"`
	Concurrency string `header:"concurrency" json:"concurrency"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt   int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.TrafficBudget
}

func (b *TrafficBudget) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(b.orig, "", "  ")
}

func (b *TrafficBudget) MarshalCSVValue() any {
	return []*TrafficBudget{b}
}

func toTrafficBudget(b *ps.TrafficBudget) *TrafficBudget {
	return &TrafficBudget{
		ID:          b.ID,
		Name:        b.Name,
		Mode:        b.Mode,
		Capacity:    formatOptionalInt(b.Capacity),
		Rate:        formatOptionalInt(b.Rate),
		Burst:       formatOptionalInt(b.Burst),
		Concurrency: formatOptionalInt(b.Concurrency),
		CreatedAt:   printer.GetMilliseconds(b.CreatedAt),
		UpdatedAt:   printer.GetMilliseconds(b.UpdatedAt),
		orig:        b,
	}
}

type TrafficRuleDisplay struct {
	ID          string `header:"id" json:"id"`
	Kind        string `header:"kind" json:"kind"`
	Fingerprint string `header:"fingerprint" json:"fingerprint"`
	Keyspace    string `header:"keyspace" json:"keyspace"`
	Tags        string `header:"tags" json:"tags"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt   int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.TrafficRule
}

func (r *TrafficRuleDisplay) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *TrafficRuleDisplay) MarshalCSVValue() any {
	return []*TrafficRuleDisplay{r}
}

func toTrafficRule(r *ps.TrafficRule) *TrafficRuleDisplay {
	return &TrafficRuleDisplay{
		ID:          r.ID,
		Kind:        r.Kind,
		Fingerprint: formatOptionalString(r.Fingerprint),
		Keyspace:    formatOptionalString(r.Keyspace),
		Tags:        formatTags(r.Tags),
		CreatedAt:   printer.GetMilliseconds(r.CreatedAt),
		UpdatedAt:   printer.GetMilliseconds(r.UpdatedAt),
		orig:        r,
	}
}

func formatOptionalInt(v *int) string {
	if v == nil {
		return "-"
	}
	return strconv.Itoa(*v)
}

func formatOptionalString(v *string) string {
	if v == nil {
		return "-"
	}
	return *v
}

func formatTags(tags []ps.TrafficRuleTag) string {
	if len(tags) == 0 {
		return "-"
	}
	parts := make([]string, len(tags))
	for i, t := range tags {
		if t.Source == "sql" {
			parts[i] = fmt.Sprintf("%s=%s", t.Key, t.Value)
		} else {
			parts[i] = fmt.Sprintf("%s=%s (%s)", t.Key, t.Value, t.Source)
		}
	}
	return strings.Join(parts, ", ")
}
