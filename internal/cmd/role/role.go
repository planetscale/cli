package role

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func RoleCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "role",
		Short:             "Manage database roles for a Postgres or Neki database branch",
		Long:              "Manage database roles for a Postgres or Neki database branch.\n\nThis command is supported for Postgres or Neki databases.",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(
		CreateCmd(ch),
		DeleteCmd(ch),
		GetCmd(ch),
		DefaultCmd(ch),
		ListCmd(ch),
		ReassignCmd(ch),
		RenewCmd(ch),
		ResetCmd(ch),
		ResetDefaultCmd(ch),
		UpdateCmd(ch),
	)

	return cmd
}
