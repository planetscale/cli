package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"

	ps "github.com/planetscale/cli/internal/planetscale"
)

func PromoteCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		clusterSize string
		replicas    int
	}

	promoteReq := &ps.PromoteRequest{}

	cmd := &cobra.Command{
		Use:   "promote <database> <branch>",
		Short: "Promote a branch to production",
		Long: `Promote a branch to production.

This command is supported for MySQL databases and for enabled Postgres branches. Neki databases are not supported.

--cluster-size sets the cluster size to use. PS-DEV is not permitted. When
omitted, a PS-DEV Postgres branch uses the smallest production size. For
MySQL, the size applies to the default keyspace. Changing the size requires
the write_database permission.

--replicas sets the Postgres replica count, not including the primary. 0 is a
single node. 2 to 8 is high availability. Omit it to leave the count
unchanged. Changing the replica count requires the write_database permission.`,
		Example: `  pscale branch promote mydb dev
  pscale branch promote mydb dev --cluster-size PS-10
  pscale branch promote mydb dev --cluster-size PS-10 --replicas 2`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			client, err := ch.Client()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			org := ch.Config.Organization // --org flag
			if org == "" {
				cfg, err := ch.ConfigFS.DefaultConfig()
				if err != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}

				org = cfg.Organization
			}

			databases, err := client.Databases.List(cmd.Context(), &ps.ListDatabasesRequest{
				Organization: org,
			})
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			candidates := make([]string, 0, len(databases))
			for _, db := range databases {
				candidates = append(candidates, db.Name)
			}

			return candidates, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db := args[0]
			branch := args[1]

			promoteReq.Database = db
			promoteReq.Organization = ch.Config.Organization
			promoteReq.Branch = branch
			promoteReq.ClusterSize = cmdutil.ToSizeSKUName(flags.clusterSize)
			if cmd.Flags().Changed("replicas") {
				replicas := flags.replicas
				promoteReq.Replicas = &replicas
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Promoting %s branch in %s to production...", printer.BoldBlue(branch), printer.BoldBlue(db)))
			defer end()
			dbBranch, err := client.DatabaseBranches.Promote(cmd.Context(), promoteReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						cmd.Context(), cmd, ch.Config, ch.Client, err, "connect_production_branch",
						"branch %s does not exist in database %s",
						printer.BoldBlue(branch), printer.BoldBlue(db))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Branch %s in %s was successfully promoted to production.\n", printer.BoldBlue(branch), printer.BoldBlue(db))
				if dbBranch.Kind == "" || dbBranch.Kind == ps.DatabaseEngineMySQL {
					ch.Printer.Printf("Note: In order to use deploy requests to deploy schema changes to this branch, safe migrations must also be enabled.\n")
				}
				return nil
			}

			return ch.Printer.PrintResource(ToDatabaseBranch(dbBranch))
		},
	}

	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "Cluster size to apply when promoting (for example, PS-10). PS-DEV is not permitted. Use 'pscale size cluster list' to see valid sizes.")
	cmd.Flags().IntVar(&flags.replicas, "replicas", 0, "Postgres replica count, not including the primary. 0 is a single node. 2 to 8 is high availability.")

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.ClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}
