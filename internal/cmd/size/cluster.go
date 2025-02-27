package size

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ClusterCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cluster <command>",
		Short:   "List the sizes for PlanetScale databases",
		Aliases: []string{"clusters"},
	}

	cmd.AddCommand(
		ListCmd(ch),
	)

	return cmd
}

func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		region string
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List the sizes that are available for a PlanetScale database",
		Args:    cobra.NoArgs,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := ch.Client()
			if err != nil {
				return err
			}

			clusterSKUs, err := client.Organizations.ListClusterSKUs(ctx, &planetscale.ListOrganizationClusterSKUsRequest{
				Organization: ch.Config.Organization,
			}, planetscale.WithRates(), planetscale.WithRegion(flags.region))
			if err != nil {
				return err
			}

			slices.SortFunc(clusterSKUs, func(a, b *planetscale.ClusterSKU) int {
				return cmp.Compare(a.SortOrder, b.SortOrder)
			})

			return ch.Printer.PrintResource(toClusterSKUs(clusterSKUs))
		},
	}

	cmd.Flags().StringVar(&flags.region, "region", "", "view cluster sizes and rates for a specific region")

	cmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.RegionsCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}

type ClusterSKU struct {
	Name         string `header:"name" json:"name"`
	Price        string `header:"cost" json:"rate"`
	ReplicaPrice string `header:"cost per extra replica" json:"replica_rate"`
	Provider     string `header:"provider,-" json:"provider"`
	InstanceType string `header:"instance type,n/a" json:"instance_type"`
	CPU          string `header:"cpu" json:"cpu"`
	Memory       string `header:"memory" json:"memory"`
	Storage      string `header:"storage,n/a" json:"storage"`

	orig *planetscale.ClusterSKU
}

func (c *ClusterSKU) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(c.orig, "", " ")
}

func (c *ClusterSKU) MarshalCSVValue() interface{} {
	return []*ClusterSKU{c}
}

func toClusterSKU(clusterSKU *planetscale.ClusterSKU) *ClusterSKU {
	storage := ""
	if clusterSKU.Storage != nil {
		storage = cmdutil.FormatParts(*clusterSKU.Storage).IntString()
	}

	cpu := fmt.Sprintf("%s vCPU", clusterSKU.CPU)
	memory := cmdutil.FormatParts(clusterSKU.Memory).IntString()
	rate := fmt.Sprintf("$%d", *clusterSKU.Rate)
	replicaRate := ""
	if clusterSKU.ReplicaRate != nil {
		replicaRate = fmt.Sprintf("$%d", *clusterSKU.ReplicaRate)
	}

	provider := ""
	if clusterSKU.Provider != nil {
		provider = *clusterSKU.Provider
	}

	instanceType := ""
	if clusterSKU.ProviderInstanceType != nil {
		instanceType = *clusterSKU.ProviderInstanceType
	}

	cluster := &ClusterSKU{
		Name:         clusterSKU.Name,
		Storage:      storage,
		CPU:          cpu,
		Provider:     provider,
		InstanceType: instanceType,
		Memory:       memory,
		Price:        rate,
		ReplicaPrice: replicaRate,
		orig:         clusterSKU,
	}

	return cluster
}

func toClusterSKUs(clusterSKUs []*planetscale.ClusterSKU) []*ClusterSKU {
	clusters := make([]*ClusterSKU, 0, len(clusterSKUs))

	for _, clusterSKU := range clusterSKUs {
		if clusterSKU.Enabled && clusterSKU.Rate != nil {
			clusters = append(clusters, toClusterSKU(clusterSKU))
		}
	}

	return clusters
}
