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
	var configProfiles []string
	var routers []string

	cmd := &cobra.Command{
		Use:   "restore <database> <new-branch> <backup>",
		Short: "Restore a backup to a new branch",
		Long: `Restore a backup to a new branch.

<new-branch> is the name of the branch to create. It must not already exist.
The backup is identified by id; the source branch is not a restore argument.
Preview Neki restore sizes from the source branch with:

  pscale backup restore show <database> <source-branch> <backup>`,
		Args: cmdutil.RequiredArgs("database", "new-branch", "backup"),
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

			if err := cmdutil.EnsureNekiRestoreSizing(db.Kind, true, false, len(configProfiles) > 0 || len(routers) > 0); err != nil {
				return err
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
				clusterName := flags.clusterSize
				if db.Kind == planetscale.DatabaseEngineNeki && !cmd.Flags().Changed("cluster-size") {
					clusterName = ""
				}

				createReq := &planetscale.CreatePostgresBranchRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Name:         branchName,
					BackupID:     backup,
					ClusterName:  clusterName,
				}
				if cmd.Flags().Changed("replicas") {
					replicas := flags.replicas
					createReq.Replicas = &replicas
				}
				if err := cmdutil.ApplyNekiRestoreSizing(createReq, configProfiles, routers); err != nil {
					return err
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

	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "PS-10", "Cluster size for restored backup branch. For Neki, omitted unless set so the source default profile size is used. Use `pscale size cluster list` to see the valid sizes.")
	cmd.Flags().IntVar(&flags.replicas, "replicas", 0, "Number of additional replicas for a PostgreSQL restore. 0 creates a single-node branch; omit to use the target cluster size default.")
	cmd.Flags().StringArrayVar(&configProfiles, "config-profile", nil, cmdutil.ConfigProfileSizeFlagHelp)
	cmd.Flags().StringArrayVar(&routers, "router", nil, cmdutil.RouterSizeFlagHelp)
	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.ClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	cmd.AddCommand(RestoreShowCmd(ch))

	return cmd
}
