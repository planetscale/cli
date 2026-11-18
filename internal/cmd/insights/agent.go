package insights

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

// AgentCmd groups the Insights Agent run and schedule commands.
func AgentCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent <command>",
		Short: "Start and review Insights Agent runs and schedules",
		Long: `The Insights Agent reviews slow queries or implements schema recommendations
against the repository connected to a database, and opens a pull request with
the changes it makes.

Runs belong to a branch. Each run reviews up to five slow query patterns or
implements up to five schema recommendations from that branch.`,
	}

	cmd.AddCommand(AgentRunsCmd(ch))
	cmd.AddCommand(AgentSchedulesCmd(ch))

	return cmd
}

func parseRunNumber(arg string) (int, error) {
	number, err := strconv.Atoi(arg)
	if err != nil || number < 1 {
		return 0, fmt.Errorf("invalid run number %q", arg)
	}
	return number, nil
}

func agentRunNotFoundError(ch *cmdutil.Helper, err error, database, branch string, number int) error {
	if cmdutil.ErrCode(err) != ps.ErrNotFound {
		return cmdutil.HandleError(err)
	}
	return fmt.Errorf("run %s does not exist on branch %s of database %s (organization: %s), or the Insights Agent is not enabled for the organization",
		printer.BoldBlue(fmt.Sprintf("#%d", number)), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
}

func agentNotFoundError(ch *cmdutil.Helper, err error, database, branch string) error {
	return featureNotFoundError(ch, err, database, branch, "the Insights Agent is not enabled for the organization")
}
