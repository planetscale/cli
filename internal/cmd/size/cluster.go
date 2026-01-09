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

			// Select the appropriate output format based on the engine filter:
			// - MySQL only: minimal columns (no engine, configuration, or replicas)
			// - PostgreSQL only: no engine column (it's implied), but has configuration and replicas
			// - All engines: full columns including engine, configuration, and replicas
			if !showAll {
				if engine == planetscale.DatabaseEngineMySQL {
					return ch.Printer.PrintResource(toMySQLClusterSKUs(allClusterSKUsWithEngine, flags.metal))
				}
				return ch.Printer.PrintResource(toPostgresClusterSKUs(allClusterSKUsWithEngine, flags.metal))
			}
			return ch.Printer.PrintResource(toGenericClusterSKUs(allClusterSKUsWithEngine, flags.metal))
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

// clusterBase holds the formatted display values common to all cluster types
type clusterBase struct {
	name, storage, cpu, memory, price, engine string
}

// formatClusterBase formats the common fields for a cluster SKU.
// If rateOverride is provided, it's used instead of the cluster's rate.
func formatClusterBase(sku *planetscale.ClusterSKU, engine planetscale.DatabaseEngine, rateOverride *int64) clusterBase {
	storage := ""
	if sku.Storage != nil {
		storage = cmdutil.FormatPartsGB(*sku.Storage).IntString()
	}

	cpu := fmt.Sprintf("%s vCPUs", sku.CPU)
	memory := cmdutil.FormatParts(sku.Memory).IntString()

	var rate int64
	if rateOverride != nil {
		rate = *rateOverride
	} else if sku.Rate != nil {
		rate = *sku.Rate
	}

	rateStr := ""
	if rate > 0 {
		rateStr = fmt.Sprintf("$%d", rate)
	}

	engineStr := "mysql"
	if engine == planetscale.DatabaseEnginePostgres {
		engineStr = "postgresql"
	}

	return clusterBase{
		name:    cmdutil.ToClusterSizeSlug(sku.Name),
		storage: storage,
		cpu:     cpu,
		memory:  memory,
		price:   rateStr,
		engine:  engineStr,
	}
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

// ClusterSKUMySQL is the minimal format for MySQL-only view (--engine mysql).
// Includes: name, cost, cpu, memory, storage (no engine, configuration, or replicas)
type ClusterSKUMySQL struct {
	Name    string `header:"name" json:"name"`
	Price   string `header:"cost" json:"rate"`
	CPU     string `header:"cpu" json:"cpu"`
	Memory  string `header:"memory" json:"memory"`
	Storage string `header:"storage,∞" json:"storage"`

	orig *planetscale.ClusterSKU
}

// ClusterSKUPostgres is the format for PostgreSQL-only view (--engine postgresql).
// Includes: name, cost, cpu, memory, storage, configuration, replicas (no engine column)
type ClusterSKUPostgres struct {
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

func (c *ClusterSKUMySQL) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(c.orig, "", " ")
}

func (c *ClusterSKUMySQL) MarshalCSVValue() interface{} {
	return []*ClusterSKUMySQL{c}
}

func (c *ClusterSKUPostgres) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(c.orig, "", " ")
}

func (c *ClusterSKUPostgres) MarshalCSVValue() interface{} {
	return []*ClusterSKUPostgres{c}
}

func toClusterSKU(sku *planetscale.ClusterSKU, engine planetscale.DatabaseEngine, configuration string, replicas string, rateOverride *int64) *ClusterSKU {
	base := formatClusterBase(sku, engine, rateOverride)
	return &ClusterSKU{
		Name:          base.name,
		Storage:       base.storage,
		CPU:           base.cpu,
		Memory:        base.memory,
		Price:         base.price,
		Engine:        base.engine,
		Configuration: configuration,
		Replicas:      replicas,
		orig:          sku,
	}
}

func toClusterSKUPostgres(sku *planetscale.ClusterSKU, configuration string, replicas string, rateOverride *int64) *ClusterSKUPostgres {
	base := formatClusterBase(sku, planetscale.DatabaseEnginePostgres, rateOverride)
	return &ClusterSKUPostgres{
		Name:          base.name,
		Storage:       base.storage,
		CPU:           base.cpu,
		Memory:        base.memory,
		Price:         base.price,
		Configuration: configuration,
		Replicas:      replicas,
		orig:          sku,
	}
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

func toClusterSKUMySQL(sku *planetscale.ClusterSKU) *ClusterSKUMySQL {
	base := formatClusterBase(sku, planetscale.DatabaseEngineMySQL, nil)
	return &ClusterSKUMySQL{
		Name:    base.name,
		Storage: base.storage,
		CPU:     base.cpu,
		Memory:  base.memory,
		Price:   base.price,
		orig:    sku,
	}
}

// toMySQLClusterSKUs converts cluster SKUs to the MySQL-specific format.
// This is the minimal view with only: name, cost, cpu, memory, storage
func toMySQLClusterSKUs(items []clusterSKUWithEngine, onlyMetal bool) []*ClusterSKUMySQL {
	clusters := make([]*ClusterSKUMySQL, 0, len(items))
	for _, item := range items {
		if !shouldIncludeCluster(item.sku, onlyMetal) {
			continue
		}
		if item.engine == planetscale.DatabaseEngineMySQL {
			clusters = append(clusters, toClusterSKUMySQL(item.sku))
		}
	}
	return clusters
}

// toPostgresClusterSKUs converts cluster SKUs to the PostgreSQL-specific format.
// Includes: name, cost, cpu, memory, storage, configuration, replicas (no engine column)
// Each cluster appears twice: once as highly available (replicas=2) and once as single node (replicas=0)
func toPostgresClusterSKUs(items []clusterSKUWithEngine, onlyMetal bool) []*ClusterSKUPostgres {
	clusters := make([]*ClusterSKUPostgres, 0, len(items)*2)

	for _, item := range items {
		if !shouldIncludeCluster(item.sku, onlyMetal) {
			continue
		}

		if item.engine != planetscale.DatabaseEnginePostgres {
			continue
		}

		// Highly available version with regular rate
		clusters = append(clusters, toClusterSKUPostgres(item.sku, "highly available", "2", nil))

		// Single node version using replica_rate (only for non-metal clusters)
		// Metal clusters can only be purchased as "highly available"
		if !item.sku.Metal && item.sku.ReplicaRate != nil {
			clusters = append(clusters, toClusterSKUPostgres(item.sku, "single node", "0", item.sku.ReplicaRate))
		}
	}

	return clusters
}

// toGenericClusterSKUs converts cluster SKUs to the full format with all columns.
// Includes: name, cost, cpu, memory, storage, engine, configuration, replicas
// PostgreSQL clusters appear twice (highly available and single node), MySQL clusters have blank configuration/replicas.
func toGenericClusterSKUs(items []clusterSKUWithEngine, onlyMetal bool) []*ClusterSKU {
	clusters := make([]*ClusterSKU, 0, len(items)*2)

	for _, item := range items {
		if !shouldIncludeCluster(item.sku, onlyMetal) {
			continue
		}

		if item.engine == planetscale.DatabaseEnginePostgres {
			// Highly available version with regular rate
			clusters = append(clusters, toClusterSKU(item.sku, item.engine, "highly available", "2", nil))

			// Single node version using replica_rate (only for non-metal clusters)
			// Metal clusters can only be purchased as "highly available"
			if !item.sku.Metal && item.sku.ReplicaRate != nil {
				clusters = append(clusters, toClusterSKU(item.sku, item.engine, "single node", "0", item.sku.ReplicaRate))
			}
		} else {
			// MySQL clusters: configuration and replicas are blank
			clusters = append(clusters, toClusterSKU(item.sku, item.engine, "", "", nil))
		}
	}

	return clusters
}
