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
			// Partitions roll up into their root so a partitioned table shows
			// as one row; materialized views are included since they also
			// consume storage.
			SQL: `
				SELECT
					schema,
					name,
					type,
					pg_size_pretty(total_bytes) AS size
				FROM (
					SELECT
						n.nspname AS schema,
						root.relname AS name,
						CASE root.relkind
							WHEN 'm' THEN 'matview'
							WHEN 'p' THEN 'partitioned table'
							ELSE 'table'
						END AS type,
						SUM(pg_total_relation_size(c.oid)) AS total_bytes
					FROM pg_class c
					JOIN pg_class root ON root.oid = COALESCE(pg_partition_root(c.oid), c.oid)
					JOIN pg_namespace n ON n.oid = root.relnamespace
					WHERE c.relkind IN ('r', 'm', 'p')
						AND n.nspname NOT IN ('pg_catalog', 'information_schema')
						AND n.nspname !~ '^pg_toast'
					GROUP BY n.nspname, root.relname, root.relkind
				) sizes
				ORDER BY total_bytes DESC
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
					t.relname || '.' || c.relname AS name,
					pg_size_pretty(pg_relation_size(c.oid)) AS size
				FROM pg_class c
				JOIN pg_index i ON i.indexrelid = c.oid
				JOIN pg_class t ON t.oid = i.indrelid
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
		Name:         "invalid-indexes",
		Short:        "Invalid indexes left over from failed concurrent index builds",
		EmptyMessage: "No invalid indexes found.",
		MySQLHint:    "Invalid indexes are a PostgreSQL concept (left over from a failed CREATE INDEX CONCURRENTLY).",
		Postgres: &engineSQL{
			SQL: `
				SELECT
					n.nspname AS schema,
					t.relname || '.' || c.relname AS name,
					pg_size_pretty(pg_relation_size(c.oid)) AS size,
					i.indisready AS ready
				FROM pg_class c
				JOIN pg_index i ON i.indexrelid = c.oid
				JOIN pg_class t ON t.oid = i.indrelid
				JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE NOT i.indisvalid
					AND n.nspname NOT IN ('pg_catalog', 'information_schema')
					AND n.nspname !~ '^pg_toast'
				ORDER BY pg_relation_size(c.oid) DESC
				LIMIT 50;`,
		},
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
			// walsenders are excluded: replication connections stay "active"
			// forever and would always appear here.
			SQL: `
				SELECT
					pid,
					(now() - query_start)::text AS duration,
					state,
					left(query, 200) AS query
				FROM pg_stat_activity
				WHERE state <> 'idle'
					AND backend_type <> 'walsender'
					AND query NOT ILIKE '%pg_stat_activity%'
					AND now() - query_start > interval '5 minutes'
				ORDER BY now() - query_start DESC
				LIMIT 100;`,
		},
	},
	{
		Name:         "locks",
		Short:        "Blocking locks and the sessions stuck behind them",
		EmptyMessage: "No blocking locks.",
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
			// Reports only the roots of blocking trees (via pg_blocking_pids)
			// with a count of sessions stuck behind each, instead of every
			// granted lock — most locks are routine and non-blocking.
			SQL: `
				WITH RECURSIVE waiters AS (
					SELECT a.pid AS blocked_pid, b.pid AS blocking_pid
					FROM pg_stat_activity a
					CROSS JOIN LATERAL unnest(pg_blocking_pids(a.pid)) AS b(pid)
					WHERE a.wait_event_type = 'Lock'
				),
				locks AS MATERIALIZED (
					SELECT * FROM pg_locks
				),
				chain AS (
					SELECT blocked_pid, blocking_pid AS root_pid
					FROM waiters
					WHERE blocking_pid NOT IN (SELECT blocked_pid FROM waiters)
					UNION
					SELECT w.blocked_pid, c.root_pid
					FROM waiters w
					JOIN chain c ON c.blocked_pid = w.blocking_pid
				)
				SELECT
					a.pid,
					a.state,
					string_agg(DISTINCT wc.relname, ', ')  AS relation,
					string_agg(DISTINCT hl.mode, ', ')     AS lock_mode,
					string_agg(DISTINCT hl.locktype, ', ') AS lock_type,
					(now() - a.query_start)::text          AS age,
					count(DISTINCT ch.blocked_pid)         AS blocked_count,
					left(a.query, 200)                     AS query
				FROM chain ch
				JOIN pg_stat_activity a ON a.pid = ch.root_pid
				LEFT JOIN locks wl ON wl.pid = ch.blocked_pid AND NOT wl.granted
				LEFT JOIN pg_class wc ON wc.oid = wl.relation
				LEFT JOIN locks hl ON hl.pid = a.pid AND hl.granted
					AND hl.locktype = wl.locktype
					AND hl.database      IS NOT DISTINCT FROM wl.database
					AND hl.relation      IS NOT DISTINCT FROM wl.relation
					AND hl.page          IS NOT DISTINCT FROM wl.page
					AND hl.tuple         IS NOT DISTINCT FROM wl.tuple
					AND hl.virtualxid    IS NOT DISTINCT FROM wl.virtualxid
					AND hl.transactionid IS NOT DISTINCT FROM wl.transactionid
					AND hl.classid       IS NOT DISTINCT FROM wl.classid
					AND hl.objid         IS NOT DISTINCT FROM wl.objid
					AND hl.objsubid      IS NOT DISTINCT FROM wl.objsubid
				GROUP BY a.pid, a.state, a.query_start, a.query
				ORDER BY blocked_count DESC, a.query_start
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
			// Statistical estimate covering both tables and indexes. The
			// index estimate is rough: it assumes the index holds all table
			// columns.
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
							hdr + 1 + count(*) FILTER (WHERE null_frac <> 0) / 8 AS nullhdr
						FROM pg_stats s, constants
						GROUP BY schemaname, tablename, hdr, ma, bs
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
				),
				index_bloat AS (
					SELECT
						bloat_info.schemaname, bloat_info.tablename, bs,
						c2.relname AS iname,
						c2.relpages AS ipages,
						COALESCE(CEIL((c2.reltuples * (datahdr - 12)) / (bs - 20::float)), 0) AS iotta
					FROM bloat_info
					JOIN pg_class cc ON cc.relname = bloat_info.tablename
					JOIN pg_namespace nn ON cc.relnamespace = nn.oid
						AND nn.nspname = bloat_info.schemaname
						AND nn.nspname NOT IN ('information_schema', 'pg_catalog')
					JOIN pg_index i ON i.indrelid = cc.oid
					JOIN pg_class c2 ON c2.oid = i.indexrelid
					WHERE cc.relkind = 'r'
				)
				SELECT
					type, schema, object_name,
					pg_size_pretty(size) AS size,
					bloat_pct,
					pg_size_pretty(raw_waste) AS waste
				FROM (
					SELECT
						'table' AS type,
						schemaname AS schema,
						tablename AS object_name,
						(bs * relpages)::bigint AS size,
						round(CASE WHEN relpages = 0 OR relpages < otta THEN 0.0
									ELSE 100 * (relpages - otta)::numeric / relpages
								END, 1) AS bloat_pct,
						(CASE WHEN relpages < otta THEN 0 ELSE bs * (relpages - otta) END)::bigint AS raw_waste
					FROM table_bloat
					UNION ALL
					SELECT
						'index' AS type,
						schemaname AS schema,
						tablename || '::' || iname AS object_name,
						(bs * ipages)::bigint AS size,
						round(CASE WHEN ipages = 0 OR ipages < iotta THEN 0.0
									ELSE 100 * (ipages - iotta)::numeric / ipages
								END, 1) AS bloat_pct,
						(CASE WHEN ipages < iotta THEN 0 ELSE bs * (ipages - iotta) END)::bigint AS raw_waste
					FROM index_bloat
				) summary
				ORDER BY raw_waste DESC, bloat_pct DESC
				LIMIT 25;`,
		},
	},
	{
		Name:         "vacuum-stats",
		Short:        "Autovacuum and autoanalyze health: last runs, dead tuples, thresholds",
		EmptyMessage: "No user tables found.",
		MySQLHint:    "Vacuum is a PostgreSQL concept; for MySQL fragmentation see: pscale inspect bloat",
		Postgres: &engineSQL{
			// Thresholds honor per-table reloptions (including
			// autovacuum_enabled=false, reported as "disabled"), not just the
			// global settings.
			SQL: `
				SELECT
					psut.schemaname AS schema,
					psut.relname AS "table",
					to_char(psut.last_vacuum, 'YYYY-MM-DD HH24:MI')      AS last_vacuum,
					to_char(psut.last_autovacuum, 'YYYY-MM-DD HH24:MI')  AS last_autovacuum,
					to_char(psut.last_analyze, 'YYYY-MM-DD HH24:MI')     AS last_analyze,
					to_char(psut.last_autoanalyze, 'YYYY-MM-DD HH24:MI') AS last_autoanalyze,
					c.reltuples::bigint AS rowcount,
					psut.n_dead_tup AS dead_rowcount,
					psut.n_mod_since_analyze AS mods_since_analyze,
					CASE
						WHEN opts.enabled = 'false' THEN 'disabled'
						WHEN opts.vac_threshold + opts.vac_scale * GREATEST(c.reltuples, 0)
							< psut.n_dead_tup THEN 'yes'
						ELSE 'no'
					END AS expect_autovacuum,
					CASE
						WHEN opts.enabled = 'false' THEN 'disabled'
						WHEN opts.an_threshold + opts.an_scale * GREATEST(c.reltuples, 0)
							< psut.n_mod_since_analyze THEN 'yes'
						ELSE 'no'
					END AS expect_autoanalyze
				FROM pg_stat_user_tables psut
				JOIN pg_class c ON c.oid = psut.relid
				CROSS JOIN LATERAL (
					SELECT
						COALESCE((SELECT split_part(o, '=', 2)::bigint
									FROM unnest(c.reloptions) AS o
									WHERE o LIKE 'autovacuum_vacuum_threshold=%'),
								current_setting('autovacuum_vacuum_threshold')::bigint)     AS vac_threshold,
						COALESCE((SELECT split_part(o, '=', 2)::numeric
									FROM unnest(c.reloptions) AS o
									WHERE o LIKE 'autovacuum_vacuum_scale_factor=%'),
								current_setting('autovacuum_vacuum_scale_factor')::numeric) AS vac_scale,
						COALESCE((SELECT split_part(o, '=', 2)::bigint
									FROM unnest(c.reloptions) AS o
									WHERE o LIKE 'autovacuum_analyze_threshold=%'),
								current_setting('autovacuum_analyze_threshold')::bigint)    AS an_threshold,
						COALESCE((SELECT split_part(o, '=', 2)::numeric
									FROM unnest(c.reloptions) AS o
									WHERE o LIKE 'autovacuum_analyze_scale_factor=%'),
								current_setting('autovacuum_analyze_scale_factor')::numeric) AS an_scale,
						COALESCE((SELECT split_part(o, '=', 2)
									FROM unnest(c.reloptions) AS o
									WHERE o LIKE 'autovacuum_enabled=%'),
								'true') AS enabled
				) opts
				ORDER BY psut.n_dead_tup DESC
				LIMIT 25;`,
		},
	},
	{
		Name:         "replication-slots",
		Short:        "Replication slots: status, WAL retention, and lag",
		EmptyMessage: "No replication slots found.",
		MySQLHint:    "Replication slots are a PostgreSQL concept; for Vitess workflows see: pscale workflow list",
		Postgres: &engineSQL{
			// retained_wal_size (since restart_lsn) and unconfirmed_wal_size
			// (since confirmed_flush_lsn) measure different failure modes;
			// safe_wal_size shows headroom before the slot is invalidated by
			// max_slot_wal_keep_size.
			SQL: `
				SELECT
					s.slot_name,
					s.slot_type,
					s.database,
					CASE
						WHEN r.state IS NOT NULL THEN r.state
						WHEN s.active THEN 'active (no walsender)'
						ELSE 'inactive'
					END AS status,
					s.temporary,
					s.wal_status,
					pg_size_pretty(s.safe_wal_size) AS safe_wal_size,
					pg_size_pretty(
						pg_wal_lsn_diff(
							CASE WHEN pg_is_in_recovery() THEN pg_last_wal_receive_lsn()
								ELSE pg_current_wal_lsn() END,
							s.restart_lsn
						)
					) AS retained_wal_size,
					CASE
						WHEN s.slot_type = 'logical' THEN
							pg_size_pretty(
								pg_wal_lsn_diff(
									CASE WHEN pg_is_in_recovery() THEN pg_last_wal_receive_lsn()
										ELSE pg_current_wal_lsn() END,
									s.confirmed_flush_lsn
								)
							)
						ELSE NULL
					END AS unconfirmed_wal_size,
					r.replay_lag AS lag_time
				FROM pg_replication_slots s
				LEFT JOIN pg_stat_replication r ON r.pid = s.active_pid
				ORDER BY pg_wal_lsn_diff(
					CASE WHEN pg_is_in_recovery() THEN pg_last_wal_receive_lsn()
						ELSE pg_current_wal_lsn() END,
					s.restart_lsn
				) DESC NULLS LAST
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
