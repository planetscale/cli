package cmdutil

import (
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/printer"
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

	client, err := ch.Client()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	clusterSKUs, err := client.Organizations.ListClusterSKUs(ctx, &ps.ListOrganizationClusterSKUsRequest{
		Organization: org,
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	clusterSizes := make([]cobra.Completion, 0)
	for _, c := range clusterSKUs {
		if c.Enabled && strings.Contains(c.Name, toComplete) {
			clusterSizes = append(clusterSizes, cobra.Completion(c.Name))
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
		if r.Enabled && strings.Contains(r.Slug, toComplete) {
			description := fmt.Sprintf("%s (%s)", printer.Bold(r.Name), r.Location)

			cobra.CompDebugln(description, true)
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
