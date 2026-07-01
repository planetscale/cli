package importcmd

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

// ImportCmd returns the import command group.
func ImportCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import external databases into PlanetScale Postgres",
		Long: `Import databases from external sources into PlanetScale Postgres.

Available sources:
  d1  Import from Cloudflare D1 using an offline SQLite export`,
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(D1Cmd(ch))

	return cmd
}
