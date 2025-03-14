package size

import (
	"encoding/json"
	"fmt"

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
		region   string
		metal    bool
		standard bool
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

			return ch.Printer.PrintResource(toClusterSKUs(clusterSKUs, flags.standard, flags.metal))
		},
	}

	cmd.Flags().StringVar(&flags.region, "region", "", "view cluster sizes and rates for a specific region")
	cmd.Flags().BoolVar(&flags.metal, "metal", false, "view cluster sizes and rates for metal clusters")
	cmd.Flags().BoolVar(&flags.standard, "standard", false, "view cluster sizes and rates for standard clusters")

	cmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.RegionsCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}

type ClusterSKU struct {
	Name    string `header:"name" json:"name"`
	Price   string `header:"cost" json:"rate"`
	CPU     string `header:"cpu" json:"cpu"`
	Memory  string `header:"memory" json:"memory"`
	Storage string `header:"storage,∞" json:"storage"`

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
		storage = cmdutil.FormatPartsGB(*clusterSKU.Storage).IntString()
	}

	cpu := fmt.Sprintf("%s vCPUs", clusterSKU.CPU)
	memory := cmdutil.FormatParts(clusterSKU.Memory).IntString()
	rate := ""
	if *clusterSKU.Rate > 0 {
		rate = fmt.Sprintf("$%d", *clusterSKU.Rate)
	}

	cluster := &ClusterSKU{
		Name:    clusterSKU.Name,
		Storage: storage,
		CPU:     cpu,
		Memory:  memory,
		Price:   rate,
		orig:    clusterSKU,
	}

	return cluster
}

func toClusterSKUs(clusterSKUs []*planetscale.ClusterSKU, filterStandard, filterMetal bool) []*ClusterSKU {
	clusters := make([]*ClusterSKU, 0, len(clusterSKUs))

	for _, clusterSKU := range clusterSKUs {
		if clusterSKU.Enabled && clusterSKU.Rate != nil && clusterSKU.Name != "PS_DEV" {
			// If these flags match, that means we just want to list all clusters.
			if filterStandard == filterMetal {
				clusters = append(clusters, toClusterSKU(clusterSKU))
			} else if filterStandard && !clusterSKU.Metal {
				clusters = append(clusters, toClusterSKU(clusterSKU))
			} else if filterMetal && clusterSKU.Metal {
				clusters = append(clusters, toClusterSKU(clusterSKU))
			}
		}
	}

	return clusters
}
