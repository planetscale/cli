package backup

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmd/branch"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func RestoreCmd(ch *cmdutil.Helper) *cobra.Command {
	var clusterSize string

	cmd := &cobra.Command{
		Use:   "restore <database> <branch> <backup>",
		Short: "Restore a backup to a new branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "backup"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branchName := args[1]
			backup := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Restoring backup %s to %s", printer.BoldBlue(backup), printer.BoldBlue(branchName)))
			defer end()
			newBranch, err := client.DatabaseBranches.Create(ctx, &planetscale.CreateDatabaseBranchRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Name:         branchName,
				BackupID:     backup,
				ClusterSize:  clusterSize,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrintResource(branch.ToDatabaseBranch(newBranch))
		},
	}

	cmd.Flags().StringVar(&clusterSize, "cluster-size", "PS-10", "Cluster size for restored backup branch. Use `pscale size cluster list` to see the valid sizes.")
	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.ClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}
