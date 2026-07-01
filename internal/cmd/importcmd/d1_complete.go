package importcmd

import (
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/import/d1"
	"github.com/planetscale/cli/internal/printer"
)

func d1CompleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		migrationID string
		force       bool
		noNotify    bool
	}

	cmd := &cobra.Command{
		Use:     "complete <database> [branch]",
		Aliases: []string{"teardown"},
		Short:   "Mark a D1 migration as complete in local state",
		Args:    d1DatabaseBranchArgs,
		Example: `  pscale import d1 complete mydb --migration-id abc123
  pscale import d1 complete mydb --migration-id abc123 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := parseDatabaseBranch(args)
			if !flags.force && ch.Printer.Format() == printer.Human {
				if err := ch.Printer.ConfirmCommand(flags.migrationID, "import d1 complete", "complete"); err != nil {
					return err
				}
			}
			client, err := ch.Client()
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("complete", err))
			}
			resp, err := d1.CompleteResponse(d1Org(ch), database, branch, flags.migrationID)
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("complete", err))
			}
			if err := d1.Complete(d1Org(ch), database, branch, flags.migrationID, d1NotifyAPI(client, flags.noNotify)); err != nil {
				return writeD1(ch, d1.ErrorResponse("complete", err))
			}
			return writeD1(ch, resp)
		},
	}

	cmd.Flags().StringVar(&flags.migrationID, "migration-id", "", "Migration ID")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&flags.noNotify, "no-notify", false, "Skip Slack notifications for this completion")
	cmd.MarkFlagRequired("migration-id")
	return cmd
}
