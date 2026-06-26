package d1

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/postgres"
	execabs "golang.org/x/sys/execabs"
)

//go:embed pgloader_transforms.lisp
var pgloaderTransformsLisp string

const (
	pgloaderBatchSize    = "20 MB"
	pgloaderDynamicSpace = "4096" // MB per pgloader process (SBCL heap cap)

	pgloaderLargeTableRowThreshold = 100_000

	// Fast profile: small/medium tables after indexes are deferred.
	pgloaderFastPrefetchRows = 25000
	pgloaderFastBatchRows    = 25000
	pgloaderFastWorkers      = 8
	pgloaderFastConcurrency  = 2

	// Conservative profile: wide rows / large tables (e.g. attachments).
	pgloaderSlowPrefetchRows = 5000
	pgloaderSlowBatchRows    = 10000
	pgloaderSlowWorkers        = 2
	pgloaderSlowConcurrency    = 1

	pgloaderLoadWorkMem             = "256MB"
	pgloaderLoadMaintenanceWorkMem  = "512MB"
	pgloaderIndexMaintenanceWorkMem = "2GB"
)

// PgloaderOptions configures pgloader execution.
type PgloaderOptions struct {
	SQLitePath string
	DestURI    string
	InputPath  string // dump path for column-level CAST rules
	WorkDir    string
	DryRun     bool
	DataOnly   bool
	// Tables loads one table per pgloader invocation when set (recommended for
	// large databases — avoids SBCL heap exhaustion from whole-catalog planning).
	Tables []string
}

type pgloaderMemoryProfile struct {
	prefetchRows int
	batchRows    int
	workers      int
	concurrency  int
}

func pgloaderProfileForTable(rowCount int) pgloaderMemoryProfile {
	if rowCount >= pgloaderLargeTableRowThreshold {
		return pgloaderMemoryProfile{
			prefetchRows: pgloaderSlowPrefetchRows,
			batchRows:    pgloaderSlowBatchRows,
			workers:      pgloaderSlowWorkers,
			concurrency:  pgloaderSlowConcurrency,
		}
	}
	return pgloaderMemoryProfile{
		prefetchRows: pgloaderFastPrefetchRows,
		batchRows:    pgloaderFastBatchRows,
		workers:      pgloaderFastWorkers,
		concurrency:  pgloaderFastConcurrency,
	}
}

// PgloaderLoadTables returns non-ORM tables in FK-safe load order.
func PgloaderLoadTables(inputPath string) ([]string, error) {
	tables, err := ParseDump(inputPath)
	if err != nil {
		return nil, err
	}
	ordered := topologicalLoadOrder(tables)
	out := make([]string, 0, len(ordered))
	for _, name := range ordered {
		if !IsORMMetadataTable(name) {
			out = append(out, name)
		}
	}
	return out, nil
}

// RunPgloader loads SQLite into PostgreSQL using pgloader.
func RunPgloader(ctx context.Context, opts PgloaderOptions) (ImportTimings, error) {
	var timings ImportTimings
	pgloader, err := FindPgloader()
	if err != nil {
		return timings, err
	}

	if opts.WorkDir == "" {
		opts.WorkDir, err = os.MkdirTemp("", "pscale-d1-pgloader-*")
		if err != nil {
			return timings, err
		}
		defer os.RemoveAll(opts.WorkDir)
	}

	tables := opts.Tables
	tableSchemas, err := ParseDump(opts.InputPath)
	if err != nil {
		return timings, err
	}
	tableByName := make(map[string]TableSchema, len(tableSchemas))
	for _, t := range tableSchemas {
		tableByName[t.Name] = t
	}

	rowCounts, _ := CountInsertRows(opts.InputPath)

	if len(tables) == 0 {
		pgStart := time.Now()
		if err := runPgloaderScript(ctx, pgloader, opts, pgloaderScriptConfig{
			dataOnly:       opts.DataOnly,
			resetSequences: true,
			profile:        pgloaderProfileForTable(0),
		}, TableSchema{}, tableSchemas); err != nil {
			return timings, err
		}
		timings.PgloaderMs = time.Since(pgStart).Milliseconds()
		return timings, nil
	}

	pgStart := time.Now()
	for i, name := range tables {
		table, ok := tableByName[name]
		if !ok {
			return timings, fmt.Errorf("pgloader table %s: not found in dump schema", name)
		}
		profile := pgloaderProfileForTable(rowCounts[name])
		fmt.Fprintf(os.Stderr, "pgloader: loading table %d/%d %s (workers=%d prefetch=%d)\n",
			i+1, len(tables), name, profile.workers, profile.prefetchRows)
		tableStart := time.Now()
		if err := runPgloaderScript(ctx, pgloader, opts, pgloaderScriptConfig{
			dataOnly:       opts.DataOnly,
			tableName:      name,
			resetSequences: true,
			profile:        profile,
		}, table, tableSchemas); err != nil {
			return timings, fmt.Errorf("pgloader table %s: %w", name, err)
		}
		timings.TableLoads = append(timings.TableLoads, TableLoadTiming{
			Table: name,
			Ms:    time.Since(tableStart).Milliseconds(),
		})
	}
	timings.PgloaderMs = time.Since(pgStart).Milliseconds()
	return timings, nil
}

type pgloaderScriptConfig struct {
	dataOnly       bool
	tableName      string
	resetSequences bool
	profile        pgloaderMemoryProfile
}

func runPgloaderScript(ctx context.Context, pgloader string, opts PgloaderOptions, cfg pgloaderScriptConfig, table TableSchema, allTables []TableSchema) error {
	loadFile := filepath.Join(opts.WorkDir, "load.load")
	if cfg.tableName != "" {
		loadFile = filepath.Join(opts.WorkDir, "load-"+cfg.tableName+".load")
	}
	castTables := allTables
	if table.Name != "" {
		castTables = []TableSchema{table}
	}
	content := buildPgloaderScript(opts.SQLitePath, opts.DestURI, cfg, castTables, allTables)
	if err := os.WriteFile(loadFile, []byte(content), 0o600); err != nil {
		return err
	}

	if opts.DryRun {
		return nil
	}

	transformsFile := filepath.Join(opts.WorkDir, "transforms.lisp")
	if err := os.WriteFile(transformsFile, []byte(pgloaderTransformsLisp), 0o600); err != nil {
		return err
	}

	cmd := execabs.CommandContext(ctx, pgloader, "--load-lisp-file", transformsFile, loadFile)
	cmd.Env = append(os.Environ(),
		"SBCL_OPTIONS=--dynamic-space-size "+pgloaderDynamicSpace,
	)
	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		return fmt.Errorf("pgloader failed: %w: %s", err, output)
	}
	if strings.Contains(output, "FATAL") || strings.Contains(output, "KABOOM") ||
		strings.Contains(output, "ERROR Error while formatting") ||
		strings.Contains(output, "ERROR The value") ||
		strings.Contains(output, "Heap exhausted") ||
		pgloaderHadErrors(output) {
		return fmt.Errorf("pgloader failed: %s", output)
	}
	fmt.Fprint(os.Stderr, output)
	return nil
}

func buildPgloaderScript(sqlitePath, destURI string, cfg pgloaderScriptConfig, castTables, allTables []TableSchema) string {
	absSQLite, _ := filepath.Abs(sqlitePath)
	src := "sqlite:///" + strings.ReplaceAll(absSQLite, " ", "%20")
	target := destURI
	if parsed, err := postgres.ParseConnectionURI(destURI); err == nil {
		target = postgres.BuildConnectionURI(parsed)
	}

	profile := cfg.profile
	if profile.workers == 0 {
		profile = pgloaderProfileForTable(0)
	}

	var b strings.Builder
	b.WriteString("LOAD DATABASE\n")
	b.WriteString("     FROM " + src + "\n")
	b.WriteString("     INTO " + target + "\n")
	b.WriteString("\n")

	if cfg.dataOnly {
		b.WriteString(" WITH data only, create no tables, create no indexes, truncate, disable triggers,\n")
		if cfg.resetSequences {
			b.WriteString("      reset sequences,\n")
		} else {
			b.WriteString("      reset no sequences,\n")
		}
		b.WriteString(fmt.Sprintf("      workers = %d, concurrency = %d,\n", profile.workers, profile.concurrency))
		b.WriteString(fmt.Sprintf("      batch rows = %d,\n", profile.batchRows))
		b.WriteString("      batch size = " + pgloaderBatchSize + ",\n")
		b.WriteString(fmt.Sprintf("      prefetch rows = %d\n", profile.prefetchRows))
	} else {
		b.WriteString(" WITH include drop, create tables, create indexes, reset sequences,\n")
		b.WriteString(fmt.Sprintf("      workers = %d, concurrency = %d,\n", profile.workers, profile.concurrency))
		b.WriteString(fmt.Sprintf("      batch rows = %d,\n", profile.batchRows))
		b.WriteString("      batch size = " + pgloaderBatchSize + ",\n")
		b.WriteString(fmt.Sprintf("      prefetch rows = %d\n", profile.prefetchRows))
	}

	if cfg.tableName != "" {
		b.WriteString("\n")
		b.WriteString(" INCLUDING ONLY TABLE NAMES LIKE " + pgloaderQuotePattern(cfg.tableName) + "\n")
	}

	appendPgloaderCasts(&b, castTables, allTables)

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(" SET work_mem to '%s', maintenance_work_mem to '%s', synchronous_commit to 'off';\n",
		pgloaderLoadWorkMem, pgloaderLoadMaintenanceWorkMem))
	return b.String()
}

func appendPgloaderCasts(b *strings.Builder, castTables, allTables []TableSchema) {
	var rules []string
	for _, table := range castTables {
		for _, col := range table.Columns {
			pgType := sqliteTypeToPostgres(col, table, allTables)
			ref := fmt.Sprintf("column %s.%s", table.Name, col.Name)
			switch pgType {
			case "BOOLEAN":
				rules = append(rules, ref+" to boolean using sqlite-int-to-boolean")
			case "TIMESTAMPTZ":
				rules = append(rules, ref+" to timestamptz using sqlite-timestamp-to-timestamp")
			case "JSONB":
				rules = append(rules, ref+" to jsonb using sqlite-text-to-jsonb")
			}
		}
	}
	if len(rules) == 0 {
		return
	}
	b.WriteString("\n CAST ")
	for i, rule := range rules {
		if i > 0 {
			b.WriteString(",\n      ")
		} else {
			b.WriteString("\n      ")
		}
		b.WriteString(rule)
	}
}

func pgloaderQuotePattern(name string) string {
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}
