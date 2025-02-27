package cmdutil

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ClusterSizesCompletionFunc(ch *Helper, cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	ctx := cmd.Context()

	org := ch.Config.Organization // --org flag
	if org == "" {
		cfg, err := ch.ConfigFS.DefaultConfig()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		org = cfg.Organization
	}

	region, _ := cmd.Flags().GetString("region")

	client, err := ch.Client()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	clusterSKUs, err := client.Organizations.ListClusterSKUs(ctx, &ps.ListOrganizationClusterSKUsRequest{
		Organization: org,
	}, ps.WithRegion(region), ps.WithRates())
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	slices.SortFunc(clusterSKUs, func(a, b *ps.ClusterSKU) int {
		return cmp.Compare(a.SortOrder, b.SortOrder)
	})

	clusterSizes := make([]cobra.Completion, 0)
	for _, c := range clusterSKUs {
		if c.Enabled && strings.Contains(c.Name, toComplete) && c.Rate != nil {
			description := fmt.Sprintf("%s", c.DisplayName)
			if *c.Rate > 0 {
				description = fmt.Sprintf("%s · $%d/month· %s vCPU · %s memory", c.DisplayName, *c.Rate, c.CPU, formatParts(c.Memory).String())

				if c.Storage != nil && *c.Storage > 0 {
					description = fmt.Sprintf("%s · $%d/month· %s vCPU · %s memory · %s storage", c.DisplayName, *c.Rate, c.CPU, formatParts(c.Memory).String(), formatParts(*c.Storage).String())
				}

			}

			clusterSizes = append(clusterSizes, cobra.CompletionWithDesc(c.Name, description))
		}
	}

	return clusterSizes, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}

func RegionsCompletionFunc(ch *Helper, cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	ctx := cmd.Context()

	client, err := ch.Client()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	regions, err := client.Regions.List(ctx, &ps.ListRegionsRequest{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	regionStrs := make([]cobra.Completion, 0)

	for _, r := range regions {
		searchTerm := strings.ToLower(fmt.Sprintf("%s %s %s %s", r.Provider, r.Name, r.Location, r.Slug))
		if r.Enabled && strings.Contains(searchTerm, strings.ToLower(toComplete)) {
			description := fmt.Sprintf("%s (%s)", r.Name, r.Location)

			regionStrs = append(regionStrs, cobra.CompletionWithDesc(r.Slug, description))
		}
	}

	return regionStrs, cobra.ShellCompDirectiveNoFileComp
}

func DatabaseCompletionFunc(ch *Helper, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
		if strings.Contains(db.Name, toComplete) {
			candidates = append(candidates, db.Name)
		}
	}

	return candidates, cobra.ShellCompDirectiveNoFileComp
}

type ByteFormat struct {
	value float64
	unit  string
}

func (b ByteFormat) String() string {
	return fmt.Sprintf("%.2f %s", b.value, b.unit)
}

func formatParts(bytes int64) ByteFormat {
	kb := int64(1024)
	mb := int64(kb * 1024)
	gb := int64(mb * 1024)
	tb := int64(gb * 1024)
	pb := int64(tb * 1024)

	if bytes < mb {
		return ByteFormat{float64(bytes / kb), "KB"}
	} else if bytes < gb {
		return ByteFormat{float64(bytes / mb), "MB"}
	} else if bytes < tb {
		return ByteFormat{float64(bytes / gb), "GB"}
	} else if bytes < pb {
		return ByteFormat{float64(bytes / tb), "TB"}
	} else {
		return ByteFormat{float64(bytes / pb), "PB"}
	}
}
