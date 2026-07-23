package inspect

// engineSQL is one engine's implementation of a check: a single read-only
// diagnostic query, plus an optional PostgreSQL extension requirement.
type engineSQL struct {
	// SQL is the read-only query to run. Result sets must be bounded (LIMIT)
	// so output stays safe for agents and terminals.
	SQL string
	// RequiresExtension names a PostgreSQL extension the query depends on.
	// When set, the runner checks pg_extension first and fails with an
	// actionable hint if it isn't installed.
	RequiresExtension string
}

// check is a named diagnostic. A nil engine entry means the check doesn't
// apply to that engine; Hint explains what to use instead.
type check struct {
	Name         string
	Short        string
	EmptyMessage string
	MySQL        *engineSQL
	Postgres     *engineSQL
	// MySQLHint / PostgresHint are shown when the check isn't available for
	// that engine.
	MySQLHint    string
	PostgresHint string
	// NextSteps are related commands (usually pscale insights) that provide
	// server-side, traffic-aware analysis of the same problem. They are shown
	// after results and in JSON output so agents know to cross-reference.
	// Occurrences of <database> and <branch> are replaced with the real args.
	NextSteps []string
}

// mysqlSystemSchemas excludes MySQL system schemas and Vitess internal
// schemas from results.
const mysqlSystemSchemas = `('information_schema', 'performance_schema', 'mysql', 'sys', '_vt')`

var checks = []check{
	{
		Name:         "table-sizes",
		Short:        "Tables by total size, largest first",
		EmptyMessage: "No user tables found.",
		MySQL: &engineSQL{
			SQL: `
				SELECT
					table_schema AS ` + "`schema`" + `,
					table_name AS name,
					ROUND((data_length + index_length) / 1024 / 1024, 2) AS size_mb,
					table_rows AS approx_rows
				FROM information_schema.tables
				WHERE table_schema NOT IN ` + mysqlSystemSchemas + `
					AND table_type = 'BASE TABLE'
				ORDER BY data_length + index_length DESC
				LIMIT 25;`,
		},
		Postgres: &engineSQL{
			SQL: `
				SELECT
					n.nspname AS schema,
					c.relname AS name,
					pg_size_pretty(pg_table_size(c.oid)) AS size
				FROM pg_class c
				LEFT JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
					AND n.nspname !~ '^pg_toast'
					AND c.relkind = 'r'
				ORDER BY pg_table_size(c.oid) DESC
				LIMIT 25;`,
		},
	},
	{
		Name:         "index-sizes",
		Short:        "Indexes by size, largest first",
		EmptyMessage: "No indexes found.",
		MySQL: &engineSQL{
			SQL: `
				SELECT
					database_name AS ` + "`schema`" + `,
					table_name AS ` + "`table`" + `,
					index_name,
					ROUND(stat_value * @@innodb_page_size / 1024 / 1024, 2) AS size_mb
				FROM mysql.innodb_index_stats
				WHERE stat_name = 'size'
					AND database_name NOT IN ('mysql', 'sys', '_vt')
				ORDER BY stat_value DESC
				LIMIT 25;`,
		},
		Postgres: &engineSQL{
			SQL: `
				SELECT
					n.nspname AS schema,
					c.relname AS name,
					pg_size_pretty(pg_relation_size(c.oid)) AS size
				FROM pg_class c
				LEFT JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
					AND n.nspname !~ '^pg_toast'
					AND c.relkind = 'i'
				ORDER BY pg_relation_size(c.oid) DESC
				LIMIT 25;`,
		},
	},
	{
		Name:         "unused-indexes",
		NextSteps:    []string{"pscale insights recommendations <database>"},
		Short:        "Indexes with little or no use — removal candidates",
		EmptyMessage: "No unused indexes detected.",
		MySQL: &engineSQL{
			SQL: `
				SELECT
					object_schema AS ` + "`schema`" + `,
					object_name AS ` + "`table`" + `,
					index_name
				FROM sys.schema_unused_indexes
				WHERE object_schema NOT IN ` + mysqlSystemSchemas + `
				ORDER BY object_schema, object_name
				LIMIT 50;`,
		},
		Postgres: &engineSQL{
			SQL: `
				SELECT
					s.schemaname || '.' || s.relname AS "table",
					s.indexrelname AS index,
					pg_size_pretty(pg_relation_size(s.indexrelid)) AS index_size,
					s.idx_scan AS index_scans
				FROM pg_stat_user_indexes s
				JOIN pg_index i ON i.indexrelid = s.indexrelid
				WHERE NOT i.indisunique
					AND NOT i.indisprimary
					AND s.idx_scan < 50
				ORDER BY pg_relation_size(s.indexrelid) DESC, s.idx_scan ASC
				LIMIT 50;`,
		},
	},
	{
		Name:         "redundant-indexes",
		NextSteps:    []string{"pscale insights recommendations <database>"},
		Short:        "Indexes made redundant by another index",
		EmptyMessage: "No redundant indexes detected.",
		MySQL: &engineSQL{
			SQL: `
				SELECT
					table_schema AS ` + "`schema`" + `,
					table_name AS ` + "`table`" + `,
					redundant_index_name,
					redundant_index_columns,
					dominant_index_name,
					dominant_index_columns
				FROM sys.schema_redundant_indexes
				WHERE table_schema NOT IN ` + mysqlSystemSchemas + `
				LIMIT 50;`,
		},
		PostgresHint: "Redundant-index detection for PostgreSQL is served by schema recommendations: pscale insights recommendations <database>",
	},
	{
		Name:         "seq-scans",
		NextSteps:    []string{"pscale insights queries <database> <branch> --sort rowsReadPerReturned"},
		Short:        "Tables receiving full-table scans",
		EmptyMessage: "No full-table scans recorded.",
		MySQL: &engineSQL{
			SQL: `
				SELECT
					object_schema AS ` + "`schema`" + `,
					object_name AS name,
					rows_full_scanned,
					latency
				FROM sys.schema_tables_with_full_table_scans
				WHERE object_schema NOT IN ` + mysqlSystemSchemas + `
				ORDER BY rows_full_scanned DESC
				LIMIT 25;`,
		},
		Postgres: &engineSQL{
			SQL: `
				SELECT
					schemaname AS schema,
					relname AS name,
					seq_scan AS count
				FROM pg_stat_user_tables
				ORDER BY seq_scan DESC
				LIMIT 25;`,
		},
	},
	{
		Name:         "long-running-queries",
		NextSteps:    []string{"pscale insights queries <database> <branch> --sort p99Latency"},
		Short:        "Queries running longer than 5 minutes",
		EmptyMessage: "No long-running queries.",
		MySQL: &engineSQL{
			SQL: `
				SELECT
					id,
					user,
					db,
					time AS seconds,
					state,
					LEFT(COALESCE(info, ''), 120) AS query
				FROM information_schema.processlist
				WHERE command NOT IN ('Sleep', 'Binlog Dump', 'Binlog Dump GTID', 'Daemon')
					AND user NOT IN ('vt_repl', 'event_scheduler')
					AND info IS NOT NULL
					AND time > 300
				ORDER BY time DESC
				LIMIT 100;`,
		},
		Postgres: &engineSQL{
			SQL: `
				SELECT
					pid,
					(now() - query_start)::text AS duration,
					state,
					query
				FROM pg_stat_activity
				WHERE state <> 'idle'
					AND query NOT ILIKE '%pg_stat_activity%'
					AND now() - query_start > interval '5 minutes'
				ORDER BY now() - query_start DESC
				LIMIT 100;`,
		},
	},
	{
		Name:         "locks",
		Short:        "Held locks and lock waits with the responsible queries",
		EmptyMessage: "No locks held.",
		MySQL: &engineSQL{
			SQL: `
				SELECT
					waiting_pid,
					LEFT(COALESCE(waiting_query, ''), 80) AS waiting_query,
					blocking_pid,
					LEFT(COALESCE(blocking_query, ''), 80) AS blocking_query,
					wait_age,
					locked_table
				FROM sys.innodb_lock_waits
				LIMIT 100;`,
		},
		Postgres: &engineSQL{
			SQL: `
				SELECT
					a.pid,
					c.relname,
					l.mode,
					l.locktype,
					l.granted,
					(now() - a.query_start)::text AS age,
					a.query
				FROM pg_locks l
				JOIN pg_stat_activity a ON a.pid = l.pid
				LEFT JOIN pg_class c ON c.oid = l.relation
				WHERE a.query <> '<insufficient privilege>'
					AND l.pid <> pg_backend_pid()
				ORDER BY a.query_start
				LIMIT 100;`,
		},
	},
	{
		Name:         "outliers",
		NextSteps:    []string{"pscale insights queries <database> <branch> --sort totalTime"},
		Short:        "Queries by cumulative execution time (needs pg_stat_statements)",
		EmptyMessage: "No statements recorded yet.",
		MySQLHint:    "Query timing for MySQL is served by insights: pscale insights queries <database> <branch> --sort totalTime",
		Postgres: &engineSQL{
			RequiresExtension: "pg_stat_statements",
			SQL: `
				SELECT
					(interval '1 millisecond' * total_exec_time)::text AS total_exec_time,
					to_char(
						(total_exec_time / nullif(sum(total_exec_time) OVER (), 0)) * 100,
						'FM990D0'
					) || '%' AS prop_exec_time,
					calls AS ncalls,
					query
				FROM pg_stat_statements s
				JOIN pg_database d ON d.oid = s.dbid
				WHERE d.datname = current_database()
				ORDER BY total_exec_time DESC
				LIMIT 25;`,
		},
	},
	{
		Name:         "calls",
		NextSteps:    []string{"pscale insights queries <database> <branch> --sort count"},
		Short:        "Most frequently called queries (needs pg_stat_statements)",
		EmptyMessage: "No statements recorded yet.",
		MySQLHint:    "Query frequency for MySQL is served by insights: pscale insights queries <database> <branch> --sort count",
		Postgres: &engineSQL{
			RequiresExtension: "pg_stat_statements",
			SQL: `
				SELECT
					calls AS ncalls,
					(interval '1 millisecond' * total_exec_time)::text AS total_exec_time,
					to_char(
						(calls / nullif(sum(calls) OVER (), 0)::float) * 100,
						'FM990D0'
					) || '%' AS prop_calls,
					query
				FROM pg_stat_statements s
				JOIN pg_database d ON d.oid = s.dbid
				WHERE d.datname = current_database()
				ORDER BY calls DESC
				LIMIT 25;`,
		},
	},
	{
		Name:         "bloat",
		NextSteps:    []string{"pscale insights recommendations <database>"},
		Short:        "Wasted space: estimated bloat (PostgreSQL) or fragmentation (MySQL)",
		EmptyMessage: "No bloat detected.",
		MySQL: &engineSQL{
			SQL: `
				SELECT
					table_schema AS ` + "`schema`" + `,
					table_name AS name,
					ROUND(data_free / 1024 / 1024, 2) AS free_mb,
					ROUND((data_length + index_length) / 1024 / 1024, 2) AS size_mb,
					ROUND(data_free / NULLIF(data_length + index_length + data_free, 0) * 100, 1) AS free_pct
				FROM information_schema.tables
				WHERE table_schema NOT IN ` + mysqlSystemSchemas + `
					AND data_free > 0
				ORDER BY data_free DESC
				LIMIT 25;`,
		},
		Postgres: &engineSQL{
			SQL: `
				WITH constants AS (
					SELECT current_setting('block_size')::numeric AS bs, 23 AS hdr, 8 AS ma
				),
				bloat_info AS (
					SELECT
						ma, bs, schemaname, tablename,
						(datawidth + (hdr + ma - (CASE WHEN hdr % ma = 0 THEN ma ELSE hdr % ma END)))::numeric AS datahdr,
						(maxfracsum * (nullhdr + ma - (CASE WHEN nullhdr % ma = 0 THEN ma ELSE nullhdr % ma END))) AS nullhdr2
					FROM (
						SELECT
							schemaname, tablename, hdr, ma, bs,
							SUM((1 - null_frac) * avg_width) AS datawidth,
							MAX(null_frac) AS maxfracsum,
							hdr + (
								SELECT 1 + count(*) / 8
								FROM pg_stats s2
								WHERE null_frac <> 0
									AND s2.schemaname = s.schemaname
									AND s2.tablename = s.tablename
							) AS nullhdr
						FROM pg_stats s, constants
						GROUP BY 1, 2, 3, 4, 5
					) AS foo
				),
				table_bloat AS (
					SELECT
						schemaname, tablename, cc.relpages, bs,
						CEIL((cc.reltuples * ((datahdr + ma
							- (CASE WHEN datahdr % ma = 0 THEN ma ELSE datahdr % ma END))
							+ nullhdr2 + 4)) / (bs - 20::float)) AS otta
					FROM bloat_info
					JOIN pg_class cc ON cc.relname = bloat_info.tablename
					JOIN pg_namespace nn ON cc.relnamespace = nn.oid
						AND nn.nspname = bloat_info.schemaname
						AND nn.nspname NOT IN ('information_schema', 'pg_catalog')
					WHERE cc.relkind = 'r'
				)
				SELECT
					'table' AS type,
					schemaname AS schema,
					tablename AS object_name,
					round(CASE WHEN otta = 0 THEN 0.0 ELSE relpages::numeric / otta::numeric END, 1) AS bloat,
					pg_size_pretty(
						(CASE WHEN relpages < otta THEN 0 ELSE (bs * (relpages - otta))::bigint END)
					) AS waste
				FROM table_bloat
				ORDER BY (CASE WHEN relpages < otta THEN 0 ELSE bs * (relpages - otta) END) DESC
				LIMIT 25;`,
		},
	},
	{
		Name:         "vacuum-stats",
		Short:        "Autovacuum health: last (auto)vacuum, dead tuples, threshold",
		EmptyMessage: "No user tables found.",
		MySQLHint:    "Vacuum is a PostgreSQL concept; for MySQL fragmentation see: pscale inspect bloat",
		Postgres: &engineSQL{
			SQL: `
				SELECT
					n.nspname AS schema,
					c.relname AS "table",
					to_char(psut.last_vacuum, 'YYYY-MM-DD HH24:MI') AS last_vacuum,
					to_char(psut.last_autovacuum, 'YYYY-MM-DD HH24:MI') AS last_autovacuum,
					c.reltuples::bigint AS rowcount,
					psut.n_dead_tup AS dead_rowcount,
					CASE
						WHEN current_setting('autovacuum_vacuum_threshold')::bigint
							+ current_setting('autovacuum_vacuum_scale_factor')::numeric
								* c.reltuples
							< psut.n_dead_tup
						THEN 'yes'
						ELSE 'no'
					END AS expect_autovacuum
				FROM pg_stat_user_tables psut
				JOIN pg_class c ON psut.relid = c.oid
				JOIN pg_namespace n ON c.relnamespace = n.oid
				ORDER BY psut.n_dead_tup DESC
				LIMIT 25;`,
		},
	},
	{
		Name:         "replication-slots",
		Short:        "Replication slots: status, client, and lag",
		EmptyMessage: "No replication slots found.",
		MySQLHint:    "Replication slots are a PostgreSQL concept; for Vitess workflows see: pscale workflow list",
		Postgres: &engineSQL{
			SQL: `
				SELECT
					s.slot_name,
					s.slot_type,
					CASE
						WHEN r.state IS NOT NULL THEN r.state
						WHEN s.active THEN 'active (no walsender)'
						ELSE 'inactive'
					END AS status,
					r.client_addr,
					s.restart_lsn,
					s.confirmed_flush_lsn,
					pg_size_pretty(
						pg_wal_lsn_diff(
							CASE
								WHEN pg_is_in_recovery() THEN pg_last_wal_receive_lsn()
								ELSE pg_current_wal_lsn()
							END,
							COALESCE(s.confirmed_flush_lsn, s.restart_lsn)
						)
					) AS replication_lag
				FROM pg_replication_slots s
				LEFT JOIN pg_stat_replication r ON r.pid = s.active_pid
				ORDER BY s.slot_name
				LIMIT 100;`,
		},
	},
	{
		Name:         "subscriptions",
		Short:        "Per-table logical replication progress on this subscriber",
		EmptyMessage: "No subscriptions found on this database.",
		MySQLHint:    "Subscriptions are a PostgreSQL concept; for imports see: pscale data-imports get",
		Postgres: &engineSQL{
			SQL: `
				SELECT
					sub.subname AS subscription,
					sr.srrelid::regclass::text AS table_name,
					CASE sr.srsubstate
						WHEN 'i' THEN 'initialize'
						WHEN 'd' THEN 'data being copied'
						WHEN 'f' THEN 'finished table copy'
						WHEN 's' THEN 'synchronized'
						WHEN 'r' THEN 'ready'
						ELSE sr.srsubstate::text
					END AS status,
					sr.srsublsn AS lsn
				FROM pg_subscription_rel sr
				JOIN pg_subscription sub ON sub.oid = sr.srsubid
				ORDER BY sub.subname, table_name
				LIMIT 100;`,
		},
	},
}
