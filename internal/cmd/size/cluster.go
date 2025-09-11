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
		region string
		metal  bool
		engine string
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List the sizes that are available for a PlanetScale database",
		Long:    "List the sizes that are available for a PlanetScale database. Use --engine to specify the database engine type.",
		Args:    cobra.NoArgs,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := ch.Client()
			if err != nil {
				return err
			}

			// Parse the engine flag
			engine, err := parseDatabaseEngine(flags.engine)
			if err != nil {
				return err
			}

			// Build the list options
			listOpts := []planetscale.ListOption{planetscale.WithRates()}
			if flags.region != "" {
				listOpts = append(listOpts, planetscale.WithRegion(flags.region))
			}

			// Add engine-specific parameter for PostgreSQL
			if engine == planetscale.DatabaseEnginePostgres {
				listOpts = append(listOpts, planetscale.WithPostgreSQL())
			}

			clusterSKUs, err := client.Organizations.ListClusterSKUs(ctx, &planetscale.ListOrganizationClusterSKUsRequest{
				Organization: ch.Config.Organization,
			}, listOpts...)
			if err != nil {
				return err
			}

			return ch.Printer.PrintResource(toClusterSKUs(clusterSKUs, flags.metal))
		},
	}

	cmd.Flags().StringVar(&flags.region, "region", "", "view cluster sizes and rates for a specific region")
	cmd.Flags().BoolVar(&flags.metal, "metal", false, "view cluster sizes and rates for clusters with metal storage")
	cmd.Flags().StringVar(&flags.engine, "engine", "mysql", "The database engine to show cluster sizes for. Supported values: mysql, postgresql. Defaults to mysql.")

	cmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return cmdutil.RegionsCompletionFunc(ch, cmd, args, toComplete)
	})

	cmd.RegisterFlagCompletionFunc("engine", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return []cobra.Completion{
			cobra.CompletionWithDesc("mysql", "A Vitess database"),
			cobra.CompletionWithDesc("postgresql", "The fastest cloud Postgres"),
		}, cobra.ShellCompDirectiveNoFileComp
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
		Name:    cmdutil.ToClusterSizeSlug(clusterSKU.Name),
		Storage: storage,
		CPU:     cpu,
		Memory:  memory,
		Price:   rate,
		orig:    clusterSKU,
	}

	return cluster
}

func parseDatabaseEngine(engine string) (planetscale.DatabaseEngine, error) {
	switch engine {
	case "mysql":
		return planetscale.DatabaseEngineMySQL, nil
	case "postgresql", "postgres":
		return planetscale.DatabaseEnginePostgres, nil
	default:
		return planetscale.DatabaseEngineMySQL, fmt.Errorf("invalid database engine %q, supported values: mysql, postgresql", engine)
	}
}

func toClusterSKUs(clusterSKUs []*planetscale.ClusterSKU, onlyMetal bool) []*ClusterSKU {
	clusters := make([]*ClusterSKU, 0, len(clusterSKUs))

	for _, clusterSKU := range clusterSKUs {
		if clusterSKU.Enabled && clusterSKU.Rate != nil && clusterSKU.Name != "PS_DEV" {
			if onlyMetal {
				if clusterSKU.Metal {
					clusters = append(clusters, toClusterSKU(clusterSKU))
				}
			} else {
				clusters = append(clusters, toClusterSKU(clusterSKU))
			}
		}
	}

	return clusters
}
