package workflow

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func WorkflowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "workflow <command>",
		Short:             "Manage the workflows for PlanetScale databases",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(VerifyDataCmd(ch))
	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(SwitchTrafficCmd(ch))
	cmd.AddCommand(ReverseTrafficCmd(ch))
	cmd.AddCommand(CutoverCmd(ch))
	cmd.AddCommand(ReverseCutoverCmd(ch))
	cmd.AddCommand(CompleteCmd(ch))
	cmd.AddCommand(CancelCmd(ch))

	return cmd
}
