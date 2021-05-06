package dumper

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/xelabs/go-mysqlstack/sqlparser/depends/common"

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

// Loader used to start the loader worker.
func (l *Loader) Run(ctx context.Context) error {
	pool, err := NewPool(l.log, l.cfg.Threads, l.cfg.Address, l.cfg.User, l.cfg.Password, l.cfg.SessionVars, "")
	if err != nil {
		return err
	}
	defer pool.Close()

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
	conn = pool.Get()
	if err := l.restoreTableSchema(l.cfg.OverwriteTables, files.schemas, conn); err != nil {
		return err
	}
	pool.Put(conn)

	// Shuffle the tables
	for i := range files.tables {
		j := rand.Intn(i + 1)
		files.tables[i], files.tables[j] = files.tables[j], files.tables[i]
	}

	var wg sync.WaitGroup
	var bytes uint64
	t := time.Now()
	for _, table := range files.tables {
		conn := pool.Get()
		wg.Add(1)
		go func(conn *Connection, table string) {
			defer func() {
				wg.Done()
				pool.Put(conn)
			}()
			r, err := l.restoreTable(table, conn)
			if err != nil {
				fmt.Printf("err = %+v\n", err)
				// TODO(fatih) log error via logger
			}

			atomic.AddUint64(&bytes, uint64(r))
		}(conn, table)
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

	wg.Wait()
	elapsed := time.Since(t)
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
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("loader.file.walk.error:%+v", err)
		}

		if !info.IsDir() {
			switch {
			case strings.HasSuffix(path, dbSuffix):
				files.databases = append(files.databases, path)
			case strings.HasSuffix(path, schemaSuffix):
				files.schemas = append(files.schemas, path)
			default:
				if strings.HasSuffix(path, tableSuffix) {
					files.tables = append(files.tables, path)
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

		data, err := ioutil.ReadFile(db)
		if err != nil {
			return err
		}

		sql := common.BytesToString(data)

		err = conn.Execute(sql)
		if err != nil {
			return err
		}

		l.log.Info("restoring database", zap.String("database", name))
	}

	return nil
}

func (l *Loader) restoreTableSchema(overwrite bool, tables []string, conn *Connection) error {
	for _, table := range tables {
		base := filepath.Base(table)
		name := strings.TrimSuffix(base, schemaSuffix)
		db := strings.Split(name, ".")[0]
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

		data, err := ioutil.ReadFile(table)
		if err != nil {
			return err
		}
		query1 := common.BytesToString(data)
		querys := strings.Split(query1, ";\n")
		for _, query := range querys {
			if !strings.HasPrefix(query, "/*") && query != "" {
				if overwrite {
					l.log.Info(
						"drop(overwrite.is.true)",
						zap.String("database", db),
						zap.String("table ", tbl),
					)

					dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s", name)
					err = conn.Execute(dropQuery)
					if err != nil {
						return err
					}
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

func (l *Loader) restoreTable(table string, conn *Connection) (int, error) {
	bytes := 0
	part := "0"
	base := filepath.Base(table)
	name := strings.TrimSuffix(base, tableSuffix)
	splits := strings.Split(name, ".")
	db := splits[0]
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

	data, err := ioutil.ReadFile(table)

	if err != nil {
		return 0, err
	}
	query1 := common.BytesToString(data)
	querys := strings.Split(query1, ";\n")
	bytes = len(query1)
	for _, query := range querys {
		if !strings.HasPrefix(query, "/*") && query != "" {
			err = conn.Execute(query)
			if err != nil {
				return 0, err
			}
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
