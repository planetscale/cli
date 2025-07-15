package role

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func RoleCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "role",
		Short:             "Manage database roles for a Postgres database branch",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.AddCommand(
		ResetDefaultCmd(ch),
	)

	return cmd
}
