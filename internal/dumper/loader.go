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

	"go.uber.org/zap"
)

// Files tuple.
type Files struct {
	databases []string
	schemas   []string
	tables    []string
}

const (
	dbSuffix     = "-schema-create.sql"
	schemaSuffix = "-schema.sql"
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
func (l *Loader) Run(ctx context.Context, ch *cmdutil.Helper) error {
	pool, err := NewPool(l.log, l.cfg.Threads, l.cfg.Address, l.cfg.User, l.cfg.Password, l.cfg.SessionVars, "")
	if err != nil {
		return err
	}
	defer pool.Close()

	if l.cfg.ShowDetails && l.cfg.AllowDifferentDestination {
		ch.Printer.Println("The allow different destination option is enabled for this restore.")
		ch.Printer.Printf("Files that do not begin with the provided database name of %s will still be processed without having to rename them first.\n", printer.BoldBlue(l.cfg.Database))
	}

	files, err := l.loadFiles(l.cfg.Outdir, ch, l.cfg.ShowDetails, l.cfg.StartFrom)
	if err != nil {
		return err
	}

	// database.
	conn := pool.Get()
	if err := l.restoreDatabaseSchema(files.databases, conn, ch, l.cfg.ShowDetails); err != nil {
		return err
	}
	pool.Put(conn)

	// tables.
	conn = pool.Get()
	if err := l.restoreTableSchema(l.cfg.OverwriteTables, files.schemas, conn, ch, l.cfg.ShowDetails, l.cfg.StartFrom); err != nil {
		return err
	}
	pool.Put(conn)

	// Commenting out to add some predictability to the order in which data files will be processed:
	// Shuffle the tables
	//for i := range files.tables {
	//	j := rand.Intn(i + 1)
	//	files.tables[i], files.tables[j] = files.tables[j], files.tables[i]
	//}

	var eg errgroup.Group
	var bytes uint64
	t := time.Now()

	numberOfDataFiles := len(files.tables)

	for idx, table := range files.tables {
		table := table
		conn := pool.Get()

		eg.Go(func() error {
			defer pool.Put(conn)

			if l.cfg.ShowDetails {
				ch.Printer.Printf("%s: %s in thread %s (File %d of %d)\n", printer.BoldGreen("Started Processing Data File"), printer.BoldBlue(filepath.Base(table)), printer.BoldBlue(conn.ID), (idx + 1), numberOfDataFiles)
			}
			fileProcessingTimeStart := time.Now()
			r, err := l.restoreTable(table, conn, ch, l.cfg.ShowDetails)

			if err != nil {
				return err
			}

			fileProcessingTimeFinish := time.Since(fileProcessingTimeStart)
			timeElapsedSofar := time.Since(t)
			if l.cfg.ShowDetails {
				ch.Printer.Printf("%s: %s in %s with %s elapsed so far (File %d of %d)\n", printer.BoldGreen("Finished Processing Data File"), printer.BoldBlue(filepath.Base(table)), printer.BoldBlue(fileProcessingTimeFinish), printer.BoldBlue(timeElapsedSofar), (idx + 1), numberOfDataFiles)
			}

			atomic.AddUint64(&bytes, uint64(r))
			return nil
		})
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

func (l *Loader) loadFiles(dir string, ch *cmdutil.Helper, showDetails bool, startFrom string) (*Files, error) {
	files := &Files{}
	if showDetails {
		ch.Printer.Println("Collecting files from folder " + printer.BoldBlue(dir))
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
				if showDetails {
					ch.Printer.Println("Database file: " + filepath.Base(path))
				}
			case strings.HasSuffix(path, schemaSuffix):
				if tbl >= startFrom {
					files.schemas = append(files.schemas, path)
					if showDetails {
						ch.Printer.Println("  |- Table file: " + printer.BoldBlue(filepath.Base(path)))
					}
				} else {
					ch.Printer.Printf("Skipping files associated with the %s table...\n", printer.BoldBlue(tbl))
				}
			default:
				if strings.HasSuffix(path, tableSuffix) {
					if tbl >= startFrom {
						files.tables = append(files.tables, path)
						if showDetails {
							ch.Printer.Println("    |- Data file: " + printer.BoldBlue(filepath.Base(path)))
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

func (l *Loader) restoreDatabaseSchema(dbs []string, conn *Connection, ch *cmdutil.Helper, showDetails bool) error {
	for _, db := range dbs {
		base := filepath.Base(db)
		name := strings.TrimSuffix(base, dbSuffix)

		data, err := os.ReadFile(db)
		if err != nil {
			return err
		}

		if showDetails {
			ch.Printer.Println("Restoring Database: " + base)
		}
		err = conn.Execute(string(data))
		if err != nil {
			return err
		}

		l.log.Info("restoring database", zap.String("database", name))
	}

	return nil
}

func (l *Loader) restoreTableSchema(overwrite bool, tables []string, conn *Connection, ch *cmdutil.Helper, showDetails bool, startFrom string) error {
	if startFrom != "" {
		ch.Printer.Printf("Starting from %s table...\n", printer.BoldBlue(startFrom))
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

		if tbl < startFrom {
			ch.Printer.Printf("Skipping %s table (Table %d of %d)...\n", printer.BoldBlue(tbl), (idx + 1), numberOfTables)
			continue
		}

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
		queries := strings.Split(query1, ";\n")
		for _, query := range queries {
			if !strings.HasPrefix(query, "/*") && query != "" {
				if overwrite {
					l.log.Info(
						"drop(overwrite.is.true)",
						zap.String("database", db),
						zap.String("table ", tbl),
					)

					if showDetails {
						ch.Printer.Println("Dropping Existing Table (if it exists): " + printer.BoldBlue(name))
					}
					dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s", name)
					err = conn.Execute(dropQuery)
					if err != nil {
						return err
					}
				}

				if showDetails {
					ch.Printer.Printf("Creating Table: %s (Table %d of %d)\n", printer.BoldBlue(name), (idx + 1), numberOfTables)
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

func (l *Loader) restoreTable(table string, conn *Connection, ch *cmdutil.Helper, showDetails bool) (int, error) {
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
	queries := strings.Split(query1, ";\n")
	lastQuery := queries[len(queries)-1]

	// Commonly for our files the last entry is non-actionable so we should exclude it automatically:
	if strings.HasPrefix(lastQuery, "/*") || lastQuery == "" {
		queries = queries[:len(queries)-1]
	}

	bytes = len(query1)
	queriesInFile := len(queries)

	for idx, query := range queries {
		if !strings.HasPrefix(query, "/*") && query != "" {
			queryBytes := len(query)
			if queryBytes <= 16777216 {
				if showDetails {
					ch.Printer.Printf("  Processing Query %s out of %s within %s in thread %s\n", printer.BoldBlue((idx + 1)), printer.BoldBlue(queriesInFile), printer.BoldBlue(base), printer.BoldBlue(conn.ID))
				}

				err = conn.Execute(query)
				if err != nil {
					return 0, err
				}
			} else {
				// Encountering this error should be uncommon for our users.
				// However, it may be encountered if users generate files manually to match our expected folder format.
				ch.Printer.Printf("%s: Query %s within %s in thread %s is larger than 16777216 bytes. Please reduce query size to avoid pkt error.\n", printer.BoldRed("ERROR"), printer.BoldBlue((idx + 1)), printer.BoldBlue(base), printer.BoldBlue(conn.ID))
				return 0, errors.New("query is larger than 16777216 bytes in size")
			}
		} else {
			ch.Printer.Printf("  Skipping Empty Query %s out of %s within %s in thread %s\n", printer.BoldBlue((idx + 1)), printer.BoldBlue(queriesInFile), printer.BoldBlue(base), printer.BoldBlue(conn.ID))
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
