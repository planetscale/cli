package keyspace

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ReadOnlyRegionsAddCmd(ch *cmdutil.Helper) *cobra.Command {
	var clusterSize string
	var replicas int

	cmd := &cobra.Command{
		Use:   "add <database> <branch> <keyspace> <region>",
		Short: "Add a read-only region to a keyspace",
		Long: "Add a read-only region to a Vitess keyspace.\n\n" +
			"<region> is a PlanetScale region slug. List available slugs with: pscale region list.",
		Args: cmdutil.RequiredArgs("database", "branch", "keyspace", "region"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, keyspace, region := args[0], args[1], args[2], args[3]

			client, current, err := getReadOnlyRegions(cmd, ch, database, branch, keyspace)
			if err != nil {
				return err
			}
			if readOnlyRegionIndex(current, region) >= 0 {
				return fmt.Errorf("read-only region %s is already configured for keyspace %s", printer.BoldBlue(region), printer.BoldBlue(keyspace))
			}

			config := &ps.ReadOnlyRegionKeyspaceConfig{Region: region}
			if cmd.Flags().Changed("cluster-size") {
				config.ClusterSize = &clusterSize
			}
			if cmd.Flags().Changed("replicas") {
				if replicas < 1 {
					return fmt.Errorf("--replicas must be greater than 0")
				}
				config.Replicas = &replicas
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Adding read-only region %s to keyspace %s", printer.BoldBlue(region), printer.BoldBlue(keyspace)))
			defer end()

			regions, err := updateReadOnlyRegions(cmd, ch, client, database, branch, keyspace, append(readOnlyRegionConfigs(current), config))
			if err != nil {
				return err
			}
			end()

			return printReadOnlyRegionMutation(ch, fmt.Sprintf("Added read-only region %s to keyspace %s.", printer.BoldBlue(region), printer.BoldBlue(keyspace)), regions)
		},
	}

	cmd.Flags().StringVar(&clusterSize, "cluster-size", "", "cluster size for the keyspace in this read-only region. Use `pscale size cluster list` to get a list of valid sizes.")
	cmd.Flags().IntVar(&replicas, "replicas", 0, "number of replicas per shard in this read-only region")
	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.BranchClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}
