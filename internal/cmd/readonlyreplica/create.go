package readonlyreplica

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// CreateCmd creates a read-only replica.
func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		region      string
		replicas    int
		clusterSize string
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch> <name>",
		Short: "Create a read-only replica",
		Long: `Create a read-only replica for a PostgreSQL database branch.

Region is required. The replica count defaults to 1 and the cluster size
defaults to the primary cluster size when those flags are omitted.`,
		Args: cmdutil.RequiredArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}
			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "read-only replicas"); err != nil {
				return err
			}

			req := &ps.CreatePostgresReadOnlyReplicaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Name:         name,
				Region:       flags.region,
				ClusterSize:  flags.clusterSize,
			}
			if cmd.Flags().Changed("replicas") {
				req.Replicas = &flags.replicas
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating read-only replica %s for %s/%s", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			replica, err := client.PostgresReadOnlyReplicas.Create(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s, branch %s, or region %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(flags.region), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Read-only replica %s is being created for %s/%s (state: %s).\n",
					printer.BoldBlue(replica.Name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(replica.State))
				return nil
			}
			return ch.Printer.PrintResource(toReadOnlyReplica(replica))
		},
	}

	cmd.Flags().StringVar(&flags.region, "region", "", "Region slug for the read-only replica")
	cmd.Flags().IntVar(&flags.replicas, "replicas", 1, "Number of instances serving reads")
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "Cluster size SKU; defaults to the primary cluster size")
	cmd.MarkFlagRequired("region") // nolint:errcheck

	cmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.RegionsCompletionFunc(ch, cmd, args, toComplete)
	})
	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.PostgresBranchClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}
