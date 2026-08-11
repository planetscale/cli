package insights

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// InsightsCmd surfaces server-side query insights for a database: aggregated
// query statistics, query errors, anomalies, and schema recommendations.
func InsightsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "insights <command>",
		Short: "Query performance insights and schema recommendations for a database",
		Long: `Surface PlanetScale's server-side analysis of a database: aggregated query
statistics (latency percentiles, rows read, errors), detected anomalies, and
schema recommendations, all computed from production traffic.

For live, connection-level diagnostics (table sizes, locks, running queries),
see pscale inspect.`,
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(QueriesCmd(ch))
	cmd.AddCommand(ErrorsCmd(ch))
	cmd.AddCommand(AnomaliesCmd(ch))
	cmd.AddCommand(TagsCmd(ch))
	cmd.AddCommand(RecommendationsCmd(ch))

	return cmd
}

// notFoundError maps a not-found API error to a message explaining both
// causes: a missing database/branch, or insights not being enabled.
func notFoundError(ch *cmdutil.Helper, err error, database, branch string) error {
	switch cmdutil.ErrCode(err) {
	case ps.ErrNotFound:
		if branch != "" {
			return fmt.Errorf("branch %s does not exist in database %s (organization: %s) or query insights is not enabled for the database",
				printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
		}
		return fmt.Errorf("database %s does not exist in organization %s",
			printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
	default:
		return cmdutil.HandleError(err)
	}
}

// truncate shortens s for table output.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
