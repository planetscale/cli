package keyspace

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ReadOnlyRegionsUpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var clusterSize string
	var replicas int

	cmd := &cobra.Command{
		Use:   "update <database> <branch> <keyspace> <region>",
		Short: "Update a keyspace's read-only region",
		Long: "Update cluster size or replicas for a keyspace's read-only region.\n\n" +
			"<region> is a PlanetScale region slug already configured on the keyspace. List configured regions with: pscale keyspace read-only-regions <database> <branch> <keyspace>.",
		Args: cmdutil.RequiredArgs("database", "branch", "keyspace", "region"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, keyspace, region := args[0], args[1], args[2], args[3]

			if !cmd.Flags().Changed("cluster-size") && !cmd.Flags().Changed("replicas") {
				return fmt.Errorf("at least one of --cluster-size or --replicas is required")
			}
			if cmd.Flags().Changed("replicas") && replicas < 1 {
				return fmt.Errorf("--replicas must be greater than 0")
			}

			client, current, err := getReadOnlyRegions(cmd, ch, database, branch, keyspace)
			if err != nil {
				return err
			}
			index := readOnlyRegionIndex(current, region)
			if index < 0 {
				return fmt.Errorf("read-only region %s is not configured for keyspace %s", printer.BoldBlue(region), printer.BoldBlue(keyspace))
			}

			configs := readOnlyRegionConfigs(current)
			if cmd.Flags().Changed("cluster-size") {
				configs[index].ClusterSize = &clusterSize
			}
			if cmd.Flags().Changed("replicas") {
				configs[index].Replicas = &replicas
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating read-only region %s for keyspace %s", printer.BoldBlue(region), printer.BoldBlue(keyspace)))
			defer end()

			regions, err := updateReadOnlyRegions(cmd, ch, client, database, branch, keyspace, configs)
			if err != nil {
				return err
			}
			end()

			return printReadOnlyRegionMutation(ch, fmt.Sprintf("Updated read-only region %s for keyspace %s.", printer.BoldBlue(region), printer.BoldBlue(keyspace)), regions)
		},
	}

	cmd.Flags().StringVar(&clusterSize, "cluster-size", "", "cluster size for the keyspace in this read-only region. Use `pscale size cluster list` to get a list of valid sizes.")
	cmd.Flags().IntVar(&replicas, "replicas", 0, "number of replicas per shard in this read-only region")
	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.BranchClusterSizesCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}
