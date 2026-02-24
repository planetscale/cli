package vtctld

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func VtctldCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "vtctld <command>",
		Short:  "Run vtctld commands against a branch",
		Long:   "Run vtctld commands against a branch. This command is only supported for Vitess databases.",
		Hidden: true,
	}

	cmd.AddCommand(VDiffCmd(ch))
	cmd.AddCommand(ListWorkflowsCmd(ch))
	cmd.AddCommand(ListKeyspacesCmd(ch))

	return cmd
}
