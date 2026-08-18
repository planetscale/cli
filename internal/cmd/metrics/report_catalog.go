package metrics

import (
	"fmt"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type reportSectionKind string

const (
	reportSeriesSection  reportSectionKind = "series"
	reportInstantSection reportSectionKind = "instant"
)

type reportSectionDefinition struct {
	Name    string
	Kind    reportSectionKind
	Metrics []string
}

var mysqlReportSections = []reportSectionDefinition{
	{
		Name: "Workload, errors, and traffic control",
		Kind: reportSeriesSection,
		Metrics: []string{
			"queries",
			"query_errors",
			"connections",
			"rows_read",
			"rows_returned",
			"rows_written",
			"violations",
			"traffic_control_warnings",
			"traffic_control_throttled",
		},
	},
	{
		Name: "Latency and execution time",
		Kind: reportSeriesSection,
		Metrics: []string{
			"latency_p50",
			"latency_p95",
			"latency_p99",
			"latency_p999",
			"latency_max",
			"vtgate_latency_p50",
			"vtgate_latency_p95",
			"total_duration_millis",
			"cpu_duration_millis",
			"io_duration_millis",
		},
	},
	{
		Name: "Query efficiency and fan-out",
		Kind: reportSeriesSection,
		Metrics: []string{
			"rows_read_per_query",
			"rows_returned_per_query",
			"rows_affected_per_query",
			"rows_read_per_returned",
			"avg_shard_queries",
			"max_shard_queries",
			"avg_parallel_workers",
		},
	},
	{
		Name: "Buffer and block activity",
		Kind: reportSeriesSection,
		Metrics: []string{
			"blocks_hit",
			"blocks_read",
			"block_cache_hit_ratio",
			"blocks_dirtied",
			"blocks_written",
		},
	},
	{
		Name: "Network traffic",
		Kind: reportSeriesSection,
		Metrics: []string{
			"ingress_bytes",
			"ingress_bytes_per_query",
			"max_ingress_bytes",
			"egress_bytes",
			"egress_bytes_per_query",
			"max_egress_bytes",
		},
	},
	{
		Name: "VTGate utilization by availability zone",
		Kind: reportSeriesSection,
		Metrics: []string{
			"vtgate_requests",
			"vtgate_cpu_by_az",
			"vtgate_cpu_avg_by_az",
			"vtgate_memory_by_az",
			"vtgate_memory_avg_by_az",
		},
	},
	{
		Name:    "Storage by table",
		Kind:    reportSeriesSection,
		Metrics: []string{"storage_per_table"},
	},
}

var postgresReportSections = []reportSectionDefinition{
	{
		Name: "Workload, errors, and traffic control",
		Kind: reportSeriesSection,
		Metrics: []string{
			"queries",
			"query_errors",
			"connections",
			"rows_read",
			"rows_returned",
			"rows_written",
			"violations",
			"traffic_control_warnings",
			"traffic_control_throttled",
		},
	},
	{
		Name: "Latency and execution time",
		Kind: reportSeriesSection,
		Metrics: []string{
			"latency_p50",
			"latency_p95",
			"latency_p99",
			"latency_p999",
			"latency_max",
			"total_duration_millis",
			"cpu_duration_millis",
			"io_duration_millis",
		},
	},
	{
		Name: "Query efficiency and distribution",
		Kind: reportSeriesSection,
		Metrics: []string{
			"rows_read_per_query",
			"rows_returned_per_query",
			"rows_affected_per_query",
			"rows_read_per_returned",
			"avg_shard_queries",
			"max_shard_queries",
			"avg_parallel_workers",
		},
	},
	{
		Name: "Buffer and block activity",
		Kind: reportSeriesSection,
		Metrics: []string{
			"blocks_hit",
			"blocks_read",
			"block_cache_hit_ratio",
			"blocks_dirtied",
			"blocks_written",
		},
	},
	{
		Name: "Query network traffic",
		Kind: reportSeriesSection,
		Metrics: []string{
			"ingress_bytes",
			"ingress_bytes_per_query",
			"max_ingress_bytes",
			"egress_bytes",
			"egress_bytes_per_query",
			"max_egress_bytes",
		},
	},
	{
		Name: "Edge network traffic",
		Kind: reportSeriesSection,
		Metrics: []string{
			"planetscale_edge_bytes_received",
			"planetscale_edge_bytes_received_rate",
			"planetscale_edge_bytes_sent",
			"planetscale_edge_bytes_sent_rate",
		},
	},
	{
		Name: "Connections and connection pooling",
		Kind: reportSeriesSection,
		Metrics: []string{
			"planetscale_dedicated_pgbouncer_current_connections",
			"planetscale_dedicated_pgbouncer_cpu_usage",
			"planetscale_dedicated_pgbouncer_memory_usage",
			"planetscale_pgbouncer_current_connections",
			"planetscale_pgbouncer_pools_client",
			"planetscale_pgbouncer_pools_server",
			"planetscale_primary_postgres_connection_state",
			"planetscale_replica_postgres_connection_state",
			"planetscale_primary_pgbouncer_cpu_util_percentages",
			"planetscale_primary_pgbouncer_mem_util_percentages",
			"planetscale_replica_pgbouncer_current_connections",
			"planetscale_replica_pgbouncer_cpu_util_percentages",
			"planetscale_replica_pgbouncer_mem_util_percentages",
		},
	},
	{
		Name: "CPU, memory utilization, and IOPS",
		Kind: reportSeriesSection,
		Metrics: []string{
			"planetscale_pods_cpu_util_percentages",
			"planetscale_pods_mem_util_percentages",
			"planetscale_pods_iops_total",
			"planetscale_primary_pods_cpu_util_percentages",
			"planetscale_primary_pods_mem_util_percentages",
			"planetscale_primary_pods_iops_total",
			"planetscale_replica_pods_cpu_util_percentages",
			"planetscale_replica_pods_mem_util_percentages",
			"planetscale_replica_pods_iops_total",
		},
	},
	{
		Name: "PostgreSQL memory composition",
		Kind: reportSeriesSection,
		Metrics: []string{
			"planetscale_primary_memory_rss_bytes",
			"planetscale_primary_memory_mmap_bytes",
			"planetscale_primary_memory_active_cache_bytes",
			"planetscale_primary_memory_inactive_cache_bytes",
			"planetscale_replica_memory_rss_bytes",
			"planetscale_replica_memory_mmap_bytes",
			"planetscale_replica_memory_active_cache_bytes",
			"planetscale_replica_memory_inactive_cache_bytes",
		},
	},
	{
		Name: "Storage utilization",
		Kind: reportSeriesSection,
		Metrics: []string{
			"planetscale_primary_storage_usage",
			"planetscale_replica_storage_usage_bytes",
			"planetscale_storage_usage_bytes",
			"planetscale_replica_volume_usage_percentages",
			"planetscale_volume_usage_percentages",
		},
	},
	{
		Name: "Transactions, replication, and WAL",
		Kind: reportSeriesSection,
		Metrics: []string{
			"planetscale_primary_xact_commit_rate",
			"planetscale_replica_lag_seconds",
			"planetscale_replication_slot_max_wal_retained_bytes",
			"planetscale_replication_slots_lost",
			"planetscale_settings_max_slot_wal_keep_size_bytes",
			"planetscale_wal_archiver_succeeded_rate",
			"planetscale_wal_archiver_failed_rate",
			"planetscale_wal_archiver_last_age_succeeded",
			"planetscale_wal_size_bytes",
		},
	},
	{
		Name: "Pod health",
		Kind: reportSeriesSection,
		Metrics: []string{
			"planetscale_pods_container_ooms",
		},
	},
	{
		Name: "Current connection capacity",
		Kind: reportInstantSection,
		Metrics: []string{
			"planetscale_dedicated_pgbouncer_current_connections",
			"planetscale_dedicated_pgbouncer_current_client_connections",
			"planetscale_dedicated_pgbouncer_current_server_connections",
			"planetscale_dedicated_pgbouncer_max_connections",
			"planetscale_dedicated_pgbouncer_cpu_usage",
			"planetscale_dedicated_pgbouncer_memory_usage",
			"planetscale_pgbouncer_current_client_connections",
			"planetscale_pgbouncer_current_server_connections",
			"planetscale_pgbouncer_settings_max_client_conn",
			"planetscale_postgres_connection_state",
			"planetscale_postgres_settings_max_connections",
		},
	},
	{
		Name: "Current storage capacity",
		Kind: reportInstantSection,
		Metrics: []string{
			"planetscale_volume_disk_usage_bytes",
			"planetscale_volume_usage_percentage",
			"planetscale_volume_capacity_bytes",
		},
	},
	{
		Name: "Backup activity",
		Kind: reportInstantSection,
		Metrics: []string{
			"planetscale_backup_restore_active",
			"planetscale_backup_fetch_percent",
		},
	},
}

func reportSectionsForEngine(engine ps.DatabaseEngine) ([]reportSectionDefinition, error) {
	switch engine {
	case ps.DatabaseEngineMySQL:
		return mysqlReportSections, nil
	case ps.DatabaseEnginePostgres:
		return postgresReportSections, nil
	default:
		return nil, fmt.Errorf("database engine %q is not supported by metrics report", engine)
	}
}
