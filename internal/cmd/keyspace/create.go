package keyspace

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// CreateCmd encapsulates the command for creating a new keyspace within a branch.
func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &planetscale.CreateBranchKeyspaceRequest{}

	var flags struct {
		shards      int
		clusterSize string
		replicas    int
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch> <keyspace>",
		Short: "Create a new keyspace within a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			createReq.Organization = ch.Config.Organization
			createReq.Database = database
			createReq.Branch = branch
			createReq.Name = keyspace
			createReq.Shards = flags.shards
			createReq.ExtraReplicas = flags.replicas
			createReq.ClusterSize = planetscale.ClusterSize(flags.clusterSize)

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating keyspace %s in %s/%s", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			k, err := client.Keyspaces.Create(ctx, createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s or branch %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Keyspace %s was successfully created.\n", printer.BoldBlue(k.Name))
				return nil
			}

			return ch.Printer.PrintResource(toBranchKeyspace(k))
		},
	}
	cmd.Flags().IntVar(&flags.shards, "shards", 1, "Number of shards in the keyspace")
	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "PS_10", "cluster size for the keyspace. Options: PS_10, PS_20, PS_40, PS_80, PS_160, PS_320, PS_400")
	cmd.Flags().IntVar(&flags.replicas, "replicas", 0, "Number of additional replicas per shard. By default, each cluster comes with 2 replicas.")

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		clusterSizes := []string{"PS_10", "PS_20", "PS_40", "PS_80", "PS_160", "PS_320", "PS_400"}

		return clusterSizes, cobra.ShellCompDirectiveDefault
	})

	return cmd
}
