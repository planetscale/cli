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
		Long:    "List the sizes that are available for a PlanetScale database. By default, shows all clusters for all engines. Use --engine to filter by a specific engine type.",
		Args:    cobra.NoArgs,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := ch.Client()
			if err != nil {
				return err
			}

			// Parse the engine flag
			engine, showAll, err := parseDatabaseEngine(flags.engine)
			if err != nil {
				return err
			}

			// Build base list options
			baseOpts := []planetscale.ListOption{planetscale.WithRates()}
			if flags.region != "" {
				baseOpts = append(baseOpts, planetscale.WithRegion(flags.region))
			}

			var allClusterSKUsWithEngine []clusterSKUWithEngine

			if showAll {
				// Make two API calls: one for MySQL, one for PostgreSQL
				// MySQL clusters (without WithPostgreSQL)
				mysqlSKUs, err := client.Organizations.ListClusterSKUs(ctx, &planetscale.ListOrganizationClusterSKUsRequest{
					Organization: ch.Config.Organization,
				}, baseOpts...)
				if err != nil {
					return cmdutil.HandleError(err)
				}
				for _, sku := range mysqlSKUs {
					allClusterSKUsWithEngine = append(allClusterSKUsWithEngine, clusterSKUWithEngine{
						sku:    sku,
						engine: planetscale.DatabaseEngineMySQL,
					})
				}

				// PostgreSQL clusters (with WithPostgreSQL)
				postgresOpts := append(baseOpts, planetscale.WithPostgreSQL())
				postgresSKUs, err := client.Organizations.ListClusterSKUs(ctx, &planetscale.ListOrganizationClusterSKUsRequest{
					Organization: ch.Config.Organization,
				}, postgresOpts...)
				if err != nil {
					return cmdutil.HandleError(err)
				}
				for _, sku := range postgresSKUs {
					allClusterSKUsWithEngine = append(allClusterSKUsWithEngine, clusterSKUWithEngine{
						sku:    sku,
						engine: planetscale.DatabaseEnginePostgres,
					})
				}
			} else {
				// Single engine filter
				listOpts := baseOpts
				if engine == planetscale.DatabaseEnginePostgres {
					listOpts = append(listOpts, planetscale.WithPostgreSQL())
				}

				clusterSKUs, err := client.Organizations.ListClusterSKUs(ctx, &planetscale.ListOrganizationClusterSKUsRequest{
					Organization: ch.Config.Organization,
				}, listOpts...)
				if err != nil {
					return cmdutil.HandleError(err)
				}
				for _, sku := range clusterSKUs {
					allClusterSKUsWithEngine = append(allClusterSKUsWithEngine, clusterSKUWithEngine{
						sku:    sku,
						engine: engine,
					})
				}
			}

			// When filtering by a single engine, omit the engine column (it's implied)
			// When showing all engines, include the engine column
			if !showAll {
				return ch.Printer.PrintResource(toClusterSKUsSingleEngine(allClusterSKUsWithEngine, flags.metal))
			}
			return ch.Printer.PrintResource(toClusterSKUs(allClusterSKUsWithEngine, flags.metal))
		},
	}

	cmd.Flags().StringVar(&flags.region, "region", "", "view cluster sizes and rates for a specific region")
	cmd.Flags().BoolVar(&flags.metal, "metal", false, "view cluster sizes and rates for clusters with metal storage")
	cmd.Flags().StringVar(&flags.engine, "engine", "", "Filter cluster sizes by database engine. Supported values: mysql, postgresql. If not specified, shows all clusters for all engines.")

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

type clusterSKUWithEngine struct {
	sku    *planetscale.ClusterSKU
	engine planetscale.DatabaseEngine
}

// ClusterSKU is the full format shown when displaying all engines (no --engine filter).
// Includes: name, cost, cpu, memory, storage, engine, configuration, replicas
type ClusterSKU struct {
	Name          string `header:"name" json:"name"`
	Price         string `header:"cost" json:"rate"`
	CPU           string `header:"cpu" json:"cpu"`
	Memory        string `header:"memory" json:"memory"`
	Storage       string `header:"storage,∞" json:"storage"`
	Engine        string `header:"engine" json:"engine"`
	Configuration string `header:"configuration" json:"configuration"`
	Replicas      string `header:"replicas" json:"replicas"`

	orig *planetscale.ClusterSKU
}

// ClusterSKUSingleEngine is the format for single-engine views (--engine mysql or --engine postgresql).
// Same as ClusterSKU but without the engine column (since it's implied by the filter).
type ClusterSKUSingleEngine struct {
	Name          string `header:"name" json:"name"`
	Price         string `header:"cost" json:"rate"`
	CPU           string `header:"cpu" json:"cpu"`
	Memory        string `header:"memory" json:"memory"`
	Storage       string `header:"storage,∞" json:"storage"`
	Configuration string `header:"configuration" json:"configuration"`
	Replicas      string `header:"replicas" json:"replicas"`

	orig *planetscale.ClusterSKU
}

func (c *ClusterSKU) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(c.orig, "", " ")
}

func (c *ClusterSKU) MarshalCSVValue() interface{} {
	return []*ClusterSKU{c}
}

func (c *ClusterSKUSingleEngine) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(c.orig, "", " ")
}

func (c *ClusterSKUSingleEngine) MarshalCSVValue() interface{} {
	return []*ClusterSKUSingleEngine{c}
}

func parseDatabaseEngine(engine string) (planetscale.DatabaseEngine, bool, error) {
	switch engine {
	case "":
		return planetscale.DatabaseEngineMySQL, true, nil // true means show all
	case "mysql":
		return planetscale.DatabaseEngineMySQL, false, nil
	case "postgresql", "postgres":
		return planetscale.DatabaseEnginePostgres, false, nil
	default:
		return planetscale.DatabaseEngineMySQL, false, fmt.Errorf("invalid database engine %q, supported values: mysql, postgresql", engine)
	}
}

// shouldIncludeCluster returns true if the cluster SKU should be included in the output.
// It filters out disabled clusters, clusters without rates, PS-DEV clusters,
// and optionally filters by metal storage.
func shouldIncludeCluster(sku *planetscale.ClusterSKU, onlyMetal bool) bool {
	if !sku.Enabled || sku.Rate == nil || sku.DisplayName == "PS-DEV" {
		return false
	}
	return !onlyMetal || sku.Metal
}

// formatClusterFields formats the common display fields for a cluster SKU.
// If rateOverride is provided, it's used instead of the cluster's rate.
func formatClusterFields(sku *planetscale.ClusterSKU, rateOverride *int64) (name, cpu, memory, storage, price string) {
	name = cmdutil.ToClusterSizeSlug(sku.Name)
	cpu = fmt.Sprintf("%s vCPUs", sku.CPU)
	memory = cmdutil.FormatParts(sku.Memory).IntString()

	if sku.Storage != nil {
		storage = cmdutil.FormatPartsGB(*sku.Storage).IntString()
	}

	var rate int64
	if rateOverride != nil {
		rate = *rateOverride
	} else if sku.Rate != nil {
		rate = *sku.Rate
	}
	if rate > 0 {
		price = fmt.Sprintf("$%d", rate)
	}

	return
}

// toClusterSKUs converts cluster SKUs to the full format with all columns including engine.
// PostgreSQL clusters appear twice (highly available and single node).
// MySQL clusters are always highly available with 2 replicas.
func toClusterSKUs(items []clusterSKUWithEngine, onlyMetal bool) []*ClusterSKU {
	clusters := make([]*ClusterSKU, 0, len(items)*2)

	for _, item := range items {
		if !shouldIncludeCluster(item.sku, onlyMetal) {
			continue
		}

		engineStr := "mysql"
		if item.engine == planetscale.DatabaseEnginePostgres {
			engineStr = "postgresql"
		}

		if item.engine == planetscale.DatabaseEnginePostgres {
			// Highly available version with regular rate
			name, cpu, memory, storage, price := formatClusterFields(item.sku, nil)
			clusters = append(clusters, &ClusterSKU{
				Name:          name,
				CPU:           cpu,
				Memory:        memory,
				Storage:       storage,
				Price:         price,
				Engine:        engineStr,
				Configuration: "highly available",
				Replicas:      "2",
				orig:          item.sku,
			})

			// Single node version using replica_rate (only for non-metal clusters)
			// Metal clusters can only be purchased as "highly available"
			if !item.sku.Metal && item.sku.ReplicaRate != nil {
				name, cpu, memory, storage, price := formatClusterFields(item.sku, item.sku.ReplicaRate)
				clusters = append(clusters, &ClusterSKU{
					Name:          name,
					CPU:           cpu,
					Memory:        memory,
					Storage:       storage,
					Price:         price,
					Engine:        engineStr,
					Configuration: "single node",
					Replicas:      "0",
					orig:          item.sku,
				})
			}
		} else {
			// MySQL clusters: always highly available with 2 replicas
			name, cpu, memory, storage, price := formatClusterFields(item.sku, nil)
			clusters = append(clusters, &ClusterSKU{
				Name:          name,
				CPU:           cpu,
				Memory:        memory,
				Storage:       storage,
				Price:         price,
				Engine:        engineStr,
				Configuration: "highly available",
				Replicas:      "2",
				orig:          item.sku,
			})
		}
	}

	return clusters
}

// toClusterSKUsSingleEngine converts cluster SKUs to the single-engine format (no engine column).
// PostgreSQL clusters appear twice (highly available and single node).
// MySQL clusters are always highly available with 2 replicas.
func toClusterSKUsSingleEngine(items []clusterSKUWithEngine, onlyMetal bool) []*ClusterSKUSingleEngine {
	clusters := make([]*ClusterSKUSingleEngine, 0, len(items)*2)

	for _, item := range items {
		if !shouldIncludeCluster(item.sku, onlyMetal) {
			continue
		}

		if item.engine == planetscale.DatabaseEnginePostgres {
			// Highly available version with regular rate
			name, cpu, memory, storage, price := formatClusterFields(item.sku, nil)
			clusters = append(clusters, &ClusterSKUSingleEngine{
				Name:          name,
				CPU:           cpu,
				Memory:        memory,
				Storage:       storage,
				Price:         price,
				Configuration: "highly available",
				Replicas:      "2",
				orig:          item.sku,
			})

			// Single node version using replica_rate (only for non-metal clusters)
			// Metal clusters can only be purchased as "highly available"
			if !item.sku.Metal && item.sku.ReplicaRate != nil {
				name, cpu, memory, storage, price := formatClusterFields(item.sku, item.sku.ReplicaRate)
				clusters = append(clusters, &ClusterSKUSingleEngine{
					Name:          name,
					CPU:           cpu,
					Memory:        memory,
					Storage:       storage,
					Price:         price,
					Configuration: "single node",
					Replicas:      "0",
					orig:          item.sku,
				})
			}
		} else {
			// MySQL clusters: always highly available with 2 replicas
			name, cpu, memory, storage, price := formatClusterFields(item.sku, nil)
			clusters = append(clusters, &ClusterSKUSingleEngine{
				Name:          name,
				CPU:           cpu,
				Memory:        memory,
				Storage:       storage,
				Price:         price,
				Configuration: "highly available",
				Replicas:      "2",
				orig:          item.sku,
			})
		}
	}

	return clusters
}
