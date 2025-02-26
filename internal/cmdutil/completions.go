package cmdutil

import (
	"fmt"
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

	clusterSizes := make([]cobra.Completion, 0)
	for _, c := range clusterSKUs {
		if c.Enabled && strings.Contains(c.Name, toComplete) && c.Rate != nil {
			description := fmt.Sprintf("%s", c.DisplayName)
			if *c.Rate > 0 {
				description = fmt.Sprintf("%s ($%d/month)", c.DisplayName, *c.Rate)
			}

			clusterSizes = append(clusterSizes, cobra.CompletionWithDesc(c.Name, description))
		}
	}

	return clusterSizes, cobra.ShellCompDirectiveNoFileComp
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
