package dumper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"golang.org/x/sync/errgroup"
	"vitess.io/vitess/go/vt/sqlparser"

	"go.uber.org/zap"
)

// Files tuple.
type Files struct {
	databases []string
	schemas   []string
	views     []string
	tables    []string
}

const (
	dbSuffix     = "-schema-create.sql"
	schemaSuffix = "-schema.sql"
	viewSuffix   = "-schema-view.sql"
	tableSuffix  = ".sql"
)

type Loader struct {
	cfg *Config
	log *zap.Logger
}

func NewLoader(cfg *Config) (*Loader, error) {
	return &Loader{
		cfg: cfg,
		log: cmdutil.NewZapLogger(cfg.Debug),
	}, nil
}

// Run used to start the loader worker.
func (l *Loader) Run(ctx context.Context) error {
	pool, err := NewPool(l.log, l.cfg.Threads, l.cfg.Address, l.cfg.User, l.cfg.Password, l.cfg.SessionVars, "")
	if err != nil {
		return err
	}
	defer pool.Close()

	if l.cfg.ShowDetails && l.cfg.AllowDifferentDestination {
		l.cfg.Printer.Println("The allow different destination option is enabled for this restore.")
		l.cfg.Printer.Printf("Files that do not begin with the provided database name of %s will still be processed without having to rename them first.\n", printer.BoldBlue(l.cfg.Database))
	}

	if l.cfg.ShowDetails && l.cfg.SchemaOnly {
		l.cfg.Printer.Println("The schema only option is enabled for this restore.")
	}

	if l.cfg.ShowDetails && l.cfg.DataOnly {
		l.cfg.Printer.Println("The data only option is enabled for this restore.")
	}

	files, err := l.loadFiles(l.cfg.Outdir)
	if err != nil {
		return err
	}

	// database.
	conn := pool.Get()
	if err := l.restoreDatabaseSchema(files.databases, conn); err != nil {
		return err
	}
	pool.Put(conn)

	// tables.
	if l.canRestoreSchema() {
		conn = pool.Get()
		if err := l.restoreTableSchema(l.cfg.OverwriteTables, files.schemas, conn); err != nil {
			return err
		}
		pool.Put(conn)
	} else {
		l.cfg.Printer.Println("Skipping restoring table definitions...")
	}

	// views.
	if l.canRestoreSchema() {
		conn = pool.Get()
		if err := l.restoreViews(l.cfg.OverwriteTables, files.views, conn); err != nil {
			return err
		}
		pool.Put(conn)
	} else {
		l.cfg.Printer.Println("Skipping restoring view definitions...")
	}

	// Adding the context here helps down below if a query issue is encountered to prevent further processing:
	eg, egCtx := errgroup.WithContext(ctx)
	var bytes uint64
	t := time.Now()

	if l.canRestoreData() {
		numberOfDataFiles := len(files.tables)

		for idx, table := range files.tables {
			// Allows for quicker exit when using Ctrl+C at the Terminal:
			if egCtx.Err() != nil {
				return egCtx.Err()
			}

			table := table
			conn := pool.Get()

			eg.Go(func() error {
				defer pool.Put(conn)

				if egCtx.Err() != nil {
					return egCtx.Err()
				}

				if l.cfg.ShowDetails {
					l.cfg.Printer.Printf("%s: %s in thread %s (File %d of %d)\n", printer.BoldGreen("Started Processing Data File"), printer.BoldBlue(filepath.Base(table)), printer.BoldBlue(conn.ID), (idx + 1), numberOfDataFiles)
				}
				fileProcessingTimeStart := time.Now()
				r, err := l.restoreTable(egCtx, table, conn)

				if err != nil {
					return err
				}

				fileProcessingTimeFinish := time.Since(fileProcessingTimeStart)
				timeElapsedSofar := time.Since(t)
				if l.cfg.ShowDetails {
					l.cfg.Printer.Printf("%s: %s in %s with %s elapsed so far (File %d of %d)\n", printer.BoldGreen("Finished Processing Data File"), printer.BoldBlue(filepath.Base(table)), printer.BoldBlue(fileProcessingTimeFinish), printer.BoldBlue(timeElapsedSofar), (idx + 1), numberOfDataFiles)
				}

				atomic.AddUint64(&bytes, uint64(r))
				return nil
			})
		}
	} else {
		l.cfg.Printer.Println("Skipping restoring data files...")
	}

	tick := time.NewTicker(time.Millisecond * time.Duration(l.cfg.IntervalMs))
	defer tick.Stop()
	go func() {
		for range tick.C {
			diff := time.Since(t).Seconds()
			bytes := float64(atomic.LoadUint64(&bytes) / 1024 / 1024)
			rates := bytes / diff
			l.log.Info(
				"restoring ...",
				zap.Float64("all_bytes", bytes),
				zap.Float64("time_diff", diff),
				zap.Float64("rates", rates),
			)
		}
	}()

	elapsed := time.Since(t)

	if err := eg.Wait(); err != nil {
		l.log.Error("error restoring", zap.Error(err))
		return err
	}

	l.log.Info(
		"restoring all done",
		zap.Duration("elapsed_time", elapsed),
		zap.Float64("all_bytes", (float64(bytes/1024/1024))),
		zap.Float64("rate_mb_seconds", (float64(bytes/1024/1024)/elapsed.Seconds())),
	)
	return nil
}

func (l *Loader) loadFiles(dir string) (*Files, error) {
	files := &Files{}
	if l.cfg.ShowDetails {
		l.cfg.Printer.Println("Collecting files from folder " + printer.BoldBlue(dir))
	}

	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("loader.file.walk.error:%+v", err)
		}

		if !info.IsDir() {
			tbl := tableNameFromFilename(path)
			switch {
			case strings.HasSuffix(path, dbSuffix):
				files.databases = append(files.databases, path)
				if l.cfg.ShowDetails {
					l.cfg.Printer.Println("Database file: " + filepath.Base(path))
				}
			case strings.HasSuffix(path, schemaSuffix):
				if l.canIncludeTable(tbl) {
					files.schemas = append(files.schemas, path)
					if l.cfg.ShowDetails {
						l.cfg.Printer.Println("  |- Table file: " + printer.BoldBlue(filepath.Base(path)))
					}
				} else {
					l.cfg.Printer.Printf("Skipping files associated with the %s table...\n", printer.BoldBlue(tbl))
				}
			case strings.HasSuffix(path, viewSuffix):
				files.views = append(files.views, path)
				if l.cfg.ShowDetails {
					l.cfg.Printer.Println("  |- View file: " + printer.BoldBlue(filepath.Base(path)))
				}
			default:
				if strings.HasSuffix(path, tableSuffix) {
					if l.canIncludeTable(tbl) {
						files.tables = append(files.tables, path)
						if l.cfg.ShowDetails {
							l.cfg.Printer.Println("    |- Data file: " + printer.BoldBlue(filepath.Base(path)))
						}
					}
				}
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("loader.file.walk.error:%+v", err)
	}
	return files, nil
}

func (l *Loader) restoreDatabaseSchema(dbs []string, conn *Connection) error {
	for _, db := range dbs {
		base := filepath.Base(db)
		name := strings.TrimSuffix(base, dbSuffix)

		data, err := os.ReadFile(db)
		if err != nil {
			return err
		}

		if l.cfg.ShowDetails {
			l.cfg.Printer.Println("Restoring Database: " + base)
		}
		err = conn.Execute(string(data))
		if err != nil {
			return err
		}

		l.log.Info("restoring database", zap.String("database", name))
	}

	return nil
}

func (l *Loader) restoreTableSchema(overwrite bool, tables []string, conn *Connection) error {
	if l.cfg.StartingTable != "" {
		l.cfg.Printer.Printf("Restore will be starting from the %s table...\n", printer.BoldBlue(l.cfg.StartingTable))
	}
	if l.cfg.EndingTable != "" {
		l.cfg.Printer.Printf("Restore will be ending at the %s table...\n", printer.BoldBlue(l.cfg.EndingTable))
	}

	numberOfTables := len(tables)

	for idx, table := range tables {
		base := filepath.Base(table)
		name := strings.TrimSuffix(base, schemaSuffix)
		db := l.databaseNameFromFilename(name)
		tbl := strings.Split(name, ".")[1]
		name = fmt.Sprintf("`%v`.`%v`", db, tbl)

		l.log.Info(
			"working table",
			zap.String("database", db),
			zap.String("table ", tbl),
		)

		err := conn.Execute(fmt.Sprintf("USE `%s`", db))
		if err != nil {
			return err
		}

		err = conn.Execute("SET FOREIGN_KEY_CHECKS=0")
		if err != nil {
			return err
		}

		data, err := os.ReadFile(table)
		if err != nil {
			return err
		}
		query1 := string(data)

		parser, err := sqlparser.New(sqlparser.Options{})
		if err != nil {
			return err
		}

		queries, err := parser.SplitStatementToPieces(query1)
		if err != nil {
			return err
		}

		for _, query := range queries {
			if !strings.HasPrefix(query, "/*") && query != "" {
				if overwrite {
					l.log.Info(
						"drop(overwrite.is.true)",
						zap.String("database", db),
						zap.String("table ", tbl),
					)

					if l.cfg.ShowDetails {
						l.cfg.Printer.Println("Dropping Existing Table (if it exists): " + printer.BoldBlue(name))
					}
					dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s", name)
					err = conn.Execute(dropQuery)
					if err != nil {
						return err
					}
				}

				if l.cfg.ShowDetails {
					l.cfg.Printer.Printf("Creating Table: %s (Table %d of %d)\n", printer.BoldBlue(name), (idx + 1), numberOfTables)
				}
				err = conn.Execute(query)
				if err != nil {
					return err
				}
			}
		}
		l.log.Info("restoring schema",
			zap.String("database", db),
			zap.String("table ", tbl),
		)
	}

	return nil
}

func (l *Loader) restoreViews(overwrite bool, views []string, conn *Connection) error {

	numberOfViews := len(views)

	for idx, viewFilename := range views {
		base := filepath.Base(viewFilename)
		name := strings.TrimSuffix(base, viewSuffix)
		db := strings.Split(name, ".")[0]
		view := strings.Split(name, ".")[1]
		name = fmt.Sprintf("`%v`.`%v`", db, view)

		l.log.Info(
			"working view",
			zap.String("database", db),
			zap.String("view ", view),
		)

		err := conn.Execute(fmt.Sprintf("USE `%s`", db))
		if err != nil {
			return err
		}

		err = conn.Execute("SET FOREIGN_KEY_CHECKS=0")
		if err != nil {
			return err
		}

		data, err := os.ReadFile(viewFilename)
		if err != nil {
			return err
		}
		query1 := string(data)

		parser, err := sqlparser.New(sqlparser.Options{})
		if err != nil {
			return err
		}

		queries, err := parser.SplitStatementToPieces(query1)
		if err != nil {
			return err
		}

		for _, query := range queries {
			if !strings.HasPrefix(query, "/*") && query != "" {
				if overwrite {
					l.log.Info(
						"drop(overwrite.is.true)",
						zap.String("database", db),
						zap.String("view ", view),
					)

					if l.cfg.ShowDetails {
						l.cfg.Printer.Println("Dropping Existing View (if it exists): " + printer.BoldBlue(name))
					}
					dropQuery := fmt.Sprintf("DROP VIEW IF EXISTS %s", name)
					err = conn.Execute(dropQuery)
					if err != nil {
						return err
					}
				}

				if l.cfg.ShowDetails {
					l.cfg.Printer.Printf("Creating View: %s (View %d of %d)\n", printer.BoldBlue(name), (idx + 1), numberOfViews)
				}
				err = conn.Execute(query)
				if err != nil {
					return err
				}
			}
		}
		l.log.Info("restoring views",
			zap.String("database", db),
			zap.String("view ", view),
		)
	}

	return nil
}

func (l *Loader) restoreTable(ctx context.Context, table string, conn *Connection) (int, error) {
	bytes := 0
	part := "0"
	base := filepath.Base(table)
	name := strings.TrimSuffix(base, tableSuffix)

	splits := strings.Split(name, ".")
	if len(splits) < 2 {
		return 0, fmt.Errorf("expected database.table, but got: %q", name)
	}

	db := l.databaseNameFromFilename(splits[0])
	tbl := splits[1]

	if len(splits) > 2 {
		part = splits[2]
	}

	l.log.Info(
		"restoring tables",
		zap.String("database", db),
		zap.String("table ", tbl),
		zap.String("part", part),
		zap.Int("thread_conn_id", conn.ID),
	)

	err := conn.Execute(fmt.Sprintf("USE `%s`", db))
	if err != nil {
		return 0, err
	}

	err = conn.Execute("SET FOREIGN_KEY_CHECKS=0")
	if err != nil {
		return 0, err
	}

	data, err := os.ReadFile(table)
	if err != nil {
		return 0, err
	}
	query1 := string(data)

	parser, err := sqlparser.New(sqlparser.Options{})
	if err != nil {
		return 0, err
	}

	queries, err := parser.SplitStatementToPieces(query1)
	if err != nil {
		return 0, err
	}

	bytes = len(query1)
	queriesInFile := len(queries)

	for idx, query := range queries {
		// Allows for quicker exit when using Ctrl+C at the Terminal:
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}

		if !strings.HasPrefix(query, "/*") && query != "" {
			queryBytes := len(query)
			if queryBytes <= l.cfg.MaxQuerySize {
				if l.cfg.ShowDetails {
					l.cfg.Printer.Printf("  Processing Query %s out of %s within %s in thread %s\n", printer.BoldBlue((idx + 1)), printer.BoldBlue(queriesInFile), printer.BoldBlue(base), printer.BoldBlue(conn.ID))
				}

				err = conn.Execute(query)
				if err != nil {
					if l.cfg.ShowDetails {
						l.cfg.Printer.Printf("  Error executing Query %s out of %s within %s in thread %s\n", printer.BoldRed((idx + 1)), printer.BoldRed(queriesInFile), printer.BoldRed(base), printer.BoldRed(conn.ID))
						l.cfg.Printer.Printf("  %s\n", printer.BoldBlack("Details:"))
						l.cfg.Printer.Printf("  %s...\n", l.substringRunes(err.Error(), 0, 512))
					}
					return 0, err
				}
			} else {
				// Encountering this error should be uncommon for our users.
				// However, it may be encountered if users generate files manually to match our expected folder format.
				l.cfg.Printer.Printf("%s: Query %s within %s in thread %s is larger than %d bytes. Please reduce query size to avoid pkt error.\n", printer.BoldRed("ERROR"), printer.BoldBlue((idx + 1)), printer.BoldBlue(base), printer.BoldBlue(conn.ID), l.cfg.MaxQuerySize)
				return 0, errors.New("query is larger than " + fmt.Sprintf("%v", l.cfg.MaxQuerySize) + " bytes in size")
			}
		} else {
			l.cfg.Printer.Printf("  Skipping Empty Query %s out of %s within %s in thread %s\n", printer.BoldBlue((idx + 1)), printer.BoldBlue(queriesInFile), printer.BoldBlue(base), printer.BoldBlue(conn.ID))
		}
	}
	l.log.Info(
		"restoring tables done...",
		zap.String("database", db),
		zap.String("table ", tbl),
		zap.String("part", part),
		zap.Int("thread_conn_id", conn.ID),
	)

	return bytes, nil
}

func (l *Loader) databaseNameFromFilename(filename string) string {
	if l.cfg.AllowDifferentDestination {
		return l.cfg.Database
	}

	return strings.Split(filename, ".")[0]
}

func (l *Loader) canIncludeTable(tbl string) bool {
	if l.cfg.StartingTable != "" && l.cfg.EndingTable != "" {
		return (tbl >= l.cfg.StartingTable && tbl <= l.cfg.EndingTable)
	}

	if l.cfg.StartingTable != "" {
		return (tbl >= l.cfg.StartingTable)
	}

	if l.cfg.EndingTable != "" {
		return (tbl <= l.cfg.EndingTable)
	}

	return true
}

func (l *Loader) canRestoreSchema() bool {
	// Default state
	if !l.cfg.SchemaOnly && !l.cfg.DataOnly {
		return true
	}

	return l.cfg.SchemaOnly
}

func (l *Loader) canRestoreData() bool {
	// Default state
	if !l.cfg.SchemaOnly && !l.cfg.DataOnly {
		return true
	}

	return l.cfg.DataOnly
}

func tableNameFromFilename(filename string) string {
	base := filepath.Base(filename)
	name := strings.TrimSuffix(base, dbSuffix)
	name = strings.TrimSuffix(name, schemaSuffix)
	name = strings.TrimSuffix(name, tableSuffix)

	splits := strings.Split(name, ".")
	if len(splits) < 2 {
		return ""
	}

	tbl := splits[1]

	return tbl
}

// https://stackoverflow.com/a/51196697
func (l *Loader) substringRunes(s string, startIndex int, count int) string {
	runes := []rune(s)
	length := len(runes)
	maxCount := length - startIndex
	if count > maxCount {
		count = maxCount
	}
	return string(runes[startIndex:count])
}
