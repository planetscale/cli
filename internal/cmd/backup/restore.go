package backup

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmd/branch"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/cli/internal/planetscale"

	"github.com/spf13/cobra"
)

func RestoreCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		clusterSize string
		replicas    int
	}

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

			db, err := client.Databases.Get(ctx, &planetscale.GetDatabaseRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Restoring backup %s to %s", printer.BoldBlue(backup), printer.BoldBlue(branchName)))
			defer end()

			if db.Kind == "mysql" {
				if cmd.Flags().Changed("replicas") {
					return fmt.Errorf("--replicas is only supported for PostgreSQL backup restores")
				}

				newBranch, err := client.DatabaseBranches.Create(ctx, &planetscale.CreateDatabaseBranchRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Name:         branchName,
					BackupID:     backup,
					ClusterSize:  flags.clusterSize,
				})
				if err != nil {
					return cmdutil.HandleError(err)
				}

				end()
				return ch.Printer.PrintResource(branch.ToDatabaseBranch(newBranch))
			} else {
				createReq := &planetscale.CreatePostgresBranchRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Name:         branchName,
					BackupID:     backup,
					ClusterName:  flags.clusterSize,
				}
				if cmd.Flags().Changed("replicas") {
					replicas := flags.replicas
					createReq.Replicas = &replicas
				}

				newBranch, err := client.PostgresBranches.Create(ctx, createReq)
				if err != nil {
					return cmdutil.HandleError(err)
				}

				end()
				return ch.Printer.PrintResource(branch.ToPostgresBranch(newBranch))
			}
		},
	}

	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "PS-10", "Cluster size for restored backup branch. Use `pscale size cluster list` to see the valid sizes.")
	cmd.Flags().IntVar(&flags.replicas, "replicas", 0, "Number of additional replicas for a PostgreSQL restore. 0 creates a single-node branch; omit to use the target cluster size default.")
	cmd.MarkFlagRequired("cluster-size")
	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.ClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}
