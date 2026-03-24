package trafficcontrol

import (
	"encoding/json"
	"strconv"

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
	)

	cmd.AddCommand(budgetCmd)
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

func formatOptionalInt(v *int) string {
	if v == nil {
		return "-"
	}
	return strconv.Itoa(*v)
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
