package backup

import (
	"fmt"

	b "github.com/planetscale/cli/internal/cmd/branch"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func RestoreCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <database> <branch> <backup>",
		Short: "Restore a backup to a new branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "backup"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			backup := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Restoring backup %s to %s", printer.BoldBlue(backup), printer.BoldBlue(branch)))
			defer end()
			newBranch, err := client.DatabaseBranches.Create(ctx, &planetscale.CreateDatabaseBranchRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Name:         branch,
				BackupID:     backup,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrintResource(b.ToDatabaseBranch(newBranch))
		},
	}

	return cmd
}
