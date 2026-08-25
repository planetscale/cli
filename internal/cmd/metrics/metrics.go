package metrics

import (
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
)

// MetricsCmd queries historical and current metrics for a database branch.
func MetricsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics <command>",
		Short: "Query historical and current metrics for a database branch",
		Long: `Query PlanetScale's metrics service for a database branch.

Human output summarizes historical series and formats current values for quick
inspection. JSON preserves the API response, while CSV emits one row per sample
or current value for use in scripts and analysis tools.`,
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(InstantCmd(ch))
	cmd.AddCommand(ReportCmd(ch))
	cmd.AddCommand(QueriesCmd(ch))
	cmd.AddCommand(TablesCmd(ch))
	cmd.AddCommand(KeyspaceTablesCmd(ch))
	cmd.AddCommand(TabletsCmd(ch))
	cmd.AddCommand(TagsCmd(ch))

	return cmd
}
