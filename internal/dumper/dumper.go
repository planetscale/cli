package dumper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	querypb "github.com/xelabs/go-mysqlstack/sqlparser/depends/query"
	"golang.org/x/sync/errgroup"

	"go.uber.org/zap"
)

const VITESS_GHOST_TABLE_REGEX = "_vt_EVAC_.*|_vt_DROP_.*|_vt_PURGE_.*|_vt_HOLD_.*|_[0-9a-zA-Z]{8}_[0-9a-zA-Z]{4}_[0-9a-zA-Z]{4}_.*"

// Config describes the settings to dump from a database.
type Config struct {
	User                      string
	Password                  string
	Address                   string
	ToUser                    string
	ToPassword                string
	ToAddress                 string
	ToDatabase                string
	ToEngine                  string
	Database                  string
	DatabaseRegexp            string
	DatabaseInvertRegexp      bool
	Shard                     string
	Table                     string
	Outdir                    string
	SessionVars               []string
	Threads                   int
	ChunksizeInMB             int
	StmtSize                  int
	Allbytes                  uint64
	Allrows                   uint64
	OverwriteTables           bool
	SchemaOnly                bool
	DataOnly                  bool
	ShowDetails               bool
	StartingTable             string
	EndingTable               string
	AllowDifferentDestination bool
	MaxQuerySize              int
	Wheres                    map[string]string
	Selects                   map[string]map[string]string
	Filters                   map[string]map[string]string

	// Interval in millisecond.
	IntervalMs int
	Debug      bool
	Printer    *printer.Printer
}

func NewDefaultConfig() *Config {
	return &Config{
		Threads: 1,
	}
}

type Dumper struct {
	cfg *Config
	log *zap.Logger
}

func NewDumper(cfg *Config) (*Dumper, error) {
	return &Dumper{
		cfg: cfg,
		log: cmdutil.NewZapLogger(cfg.Debug),
	}, nil
}

func (d *Dumper) Run(ctx context.Context) error {
	initPool, err := NewPool(d.log, d.cfg.Threads, d.cfg.Address, d.cfg.User, d.cfg.Password, nil, "")
	if err != nil {
		return err
	}
	defer initPool.Close()

	// Meta data.
	err = writeMetaData(d.cfg.Outdir)
	if err != nil {
		return err
	}

	// database.
	conn := initPool.Get()
	var databases []string
	t := time.Now()
	if d.cfg.DatabaseRegexp != "" {
		r := regexp.MustCompile(d.cfg.DatabaseRegexp)
		databases, err = d.filterDatabases(conn, r, d.cfg.DatabaseInvertRegexp)
		if err != nil {
			return err
		}
	} else {
		if d.cfg.Database != "" {
			databases = strings.Split(d.cfg.Database, ",")
		} else {
			databases, err = d.allDatabases(conn)
			if err != nil {
				return err
			}
		}
	}

	tables := make([][]string, len(databases))
	views := make([]map[string]bool, len(databases))
	for i, database := range databases {
		if d.cfg.Table != "" {
			tables[i] = strings.Split(d.cfg.Table, ",")
		} else {
			tables[i], err = d.allTables(conn, database)

			if err != nil {
				return err
			}

			views[i], err = d.allViews(conn, database)

			if err != nil {
				return err
			}
		}
	}
	initPool.Put(conn)

	// Adding the context here helps down below if a query issue is encountered to prevent further processing:
	eg, egCtx := errgroup.WithContext(ctx)
	for i, database := range databases {
		pool, err := NewPool(d.log, d.cfg.Threads/len(databases), d.cfg.Address, d.cfg.User, d.cfg.Password, d.cfg.SessionVars, database)
		if err != nil {
			return err
		}

		defer pool.Close()
		for _, table := range tables[i] {
			// Skip vitess ghost tables
			if regexp.MustCompile(VITESS_GHOST_TABLE_REGEX).MatchString(table) {
				continue
			}

			// Allows for quicker exit when using Ctrl+C at the Terminal:
			if egCtx.Err() != nil {
				return egCtx.Err()
			}

			conn := initPool.Get()
			err := d.dumpTableSchema(conn, database, table, views[i])
			if err != nil {
				return err
			}

			initPool.Put(conn)

			if _, ok := views[i][table]; ok {
				// If we just processed a view we don't want to dump it so the next part is skipped:
				continue
			}

			// Skip data dumping if schema-only mode is enabled
			if d.cfg.SchemaOnly {
				continue
			}

			conn = pool.Get()

			eg.Go(func() error {
				defer pool.Put(conn)

				if egCtx.Err() != nil {
					return egCtx.Err()
				}

				d.log.Info(
					"dumping table ...",
					zap.String("database", database),
					zap.String("table", table),
					zap.Int("thread_conn_id", conn.ID),
				)

				err := d.dumpTable(ctx, conn, database, table)
				if err != nil {
					d.log.Error("error dumping table", zap.Error(err))
				}

				return nil
			})
		}
	}

	tick := time.NewTicker(time.Millisecond * time.Duration(d.cfg.IntervalMs))
	defer tick.Stop()
	go func() {
		for range tick.C {
			diff := time.Since(t).Seconds()
			allbytesMB := float64(atomic.LoadUint64(&d.cfg.Allbytes) / 1024 / 1024)
			allrows := atomic.LoadUint64(&d.cfg.Allrows)
			rates := allbytesMB / diff
			d.log.Info(
				"dumping rates ...",
				zap.Float64("allbytes", allbytesMB),
				zap.Uint64("allrows", allrows),
				zap.Float64("time_sec", diff),
				zap.Float64("rates_mb_sec", rates),
			)
		}
	}()

	elapsed := time.Since(t)

	if err := eg.Wait(); err != nil {
		d.log.Error("error dumping", zap.Error(err))
		return err
	}

	d.log.Info(
		"dumping all done",
		zap.Duration("elapsed_time", elapsed),
		zap.Uint64("allrows", d.cfg.Allrows),
		zap.Uint64("allbytes", d.cfg.Allbytes),
		zap.Float64("rate_mb_seconds", (float64(d.cfg.Allbytes/1024/1024)/elapsed.Seconds())),
	)
	return nil
}

func writeMetaData(outdir string) error {
	file := fmt.Sprintf("%s/metadata", outdir)
	return writeFile(file, "")
}

func (d *Dumper) dumpTableSchema(conn *Connection, database string, table string, views map[string]bool) error {
	qr, err := conn.Fetch(fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", database, table))
	if err != nil {
		return err
	}

	schema := qr.Rows[0][1].String() + ";\n"

	file := fmt.Sprintf("%s/%s.%s-schema.sql", d.cfg.Outdir, database, table)
	if _, ok := views[table]; ok {
		// https://github.com/mydumper/mydumper/blob/e55612616d17281a45eed0a60a9b054cdd1fe064/src/myloader_common.c#L374
		file = fmt.Sprintf("%s/%s.%s-schema-view.sql", d.cfg.Outdir, database, table)
	}

	err = writeFile(file, schema)
	if err != nil {
		return err
	}

	d.log.Info("dumping table database schema...", zap.String("database", database), zap.String("table", table))
	return nil
}

// Dump a table in "MySQL" (multi-inserts) format
func (d *Dumper) dumpTable(ctx context.Context, conn *Connection, database string, table string) error {
	var allBytes uint64
	var allRows uint64
	var where string
	var selfields []string

	fields := make([]string, 0)
	{
		flds, err := d.dumpableFieldNames(conn, table)
		if err != nil {
			return err
		}

		for _, name := range flds {
			d.log.Debug("dump", zap.Any("filters", d.cfg.Filters), zap.String("table", table), zap.String("field_name", name))

			if _, ok := d.cfg.Filters[table][name]; ok {
				continue
			}

			fields = append(fields, fmt.Sprintf("`%s`", name))
			replacement, ok := d.cfg.Selects[table][name]
			if ok {
				selfields = append(selfields, fmt.Sprintf("%s AS `%s`", replacement, name))
			} else {
				selfields = append(selfields, fmt.Sprintf("`%s`", name))
			}
		}
	}

	if v, ok := d.cfg.Wheres[table]; ok {
		where = fmt.Sprintf(" WHERE %v", v)
	}

	cursor, err := conn.StreamFetch(fmt.Sprintf("SELECT %s FROM `%s`.`%s` %s", strings.Join(selfields, ", "), database, table, where))
	if err != nil {
		return err
	}

	fileNo := 1
	stmtsize := 0
	chunkbytes := 0
	rows := make([]string, 0, 256)
	inserts := make([]string, 0, 256)
	for cursor.Next() {
		row, err := cursor.RowValues()
		if err != nil {
			return err
		}

		// Allows for quicker exit when using Ctrl+C at the Terminal:
		if ctx.Err() != nil {
			return ctx.Err()
		}

		values := make([]string, 0, 16)
		for _, v := range row {
			if v.Raw() == nil {
				values = append(values, "NULL")
			} else {
				str := v.String()
				switch {
				case v.IsSigned(), v.IsUnsigned(), v.IsFloat(), v.IsIntegral(), v.Type() == querypb.Type_DECIMAL:
					values = append(values, str)
				default:
					values = append(values, fmt.Sprintf("\"%s\"", escapeBytes(v.Raw())))
				}
			}
		}
		r := "(" + strings.Join(values, ",") + ")"
		rows = append(rows, r)

		allRows++
		stmtsize += len(r)
		chunkbytes += len(r)
		allBytes += uint64(len(r))
		atomic.AddUint64(&d.cfg.Allbytes, uint64(len(r)))
		atomic.AddUint64(&d.cfg.Allrows, 1)

		if stmtsize >= d.cfg.StmtSize {
			insertone := fmt.Sprintf("INSERT INTO `%s`(%s) VALUES\n%s", table, strings.Join(fields, ","), strings.Join(rows, ",\n"))
			inserts = append(inserts, insertone)
			rows = rows[:0]
			stmtsize = 0
		}

		if (chunkbytes / 1024 / 1024) >= d.cfg.ChunksizeInMB {
			query := strings.Join(inserts, ";\n") + ";\n"
			file := fmt.Sprintf("%s/%s.%s.%05d.sql", d.cfg.Outdir, database, table, fileNo)
			err = writeFile(file, query)
			if err != nil {
				return err
			}

			d.log.Info(
				"dumping table ...",
				zap.String("database", database),
				zap.String("table", table),
				zap.Uint64("rows", allRows),
				zap.Any("bytes_mb", (allBytes/1024/1024)),
				zap.Int("part", fileNo),
				zap.Int("thread_conn_id", conn.ID),
			)

			inserts = inserts[:0]
			chunkbytes = 0
			fileNo++
		}
	}
	if chunkbytes > 0 {
		if len(rows) > 0 {
			insertone := fmt.Sprintf("INSERT INTO `%s`(%s) VALUES\n%s", table, strings.Join(fields, ","), strings.Join(rows, ",\n"))
			inserts = append(inserts, insertone)
		}

		query := strings.Join(inserts, ";\n") + ";\n"
		file := fmt.Sprintf("%s/%s.%s.%05d.sql", d.cfg.Outdir, database, table, fileNo)
		err = writeFile(file, query)
		if err != nil {
			return err
		}
	}
	err = cursor.Close()
	if err != nil {
		return err
	}

	d.log.Info(
		"dumping table done...",
		zap.String("database", database),
		zap.String("table", table),
		zap.Uint64("all_rows", allRows),
		zap.Any("all_bytes", (allBytes/1024/1024)),
		zap.Int("thread_conn_id", conn.ID),
	)
	return nil
}

func (d *Dumper) allTables(conn *Connection, database string) ([]string, error) {
	qr, err := conn.Fetch(fmt.Sprintf("SHOW TABLES FROM `%s`", database))
	if err != nil {
		return nil, err
	}

	tables := make([]string, 0, 128)
	for _, t := range qr.Rows {
		tables = append(tables, t[0].String())
	}
	return tables, nil
}

func (d *Dumper) allViews(conn *Connection, database string) (map[string]bool, error) {
	query := `SELECT TABLE_NAME 
			 FROM information_schema.TABLES 
			 WHERE TABLE_SCHEMA LIKE '%s' 
			 AND TABLE_TYPE = 'VIEW'
			`
	qr, err := conn.Fetch(fmt.Sprintf(query, database))
	if err != nil {
		return nil, err
	}

	views := make(map[string]bool)
	for _, t := range qr.Rows {
		views[t[0].String()] = true
	}
	return views, nil
}

func (d *Dumper) allDatabases(conn *Connection) ([]string, error) {
	qr, err := conn.Fetch("SHOW DATABASES")
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0, 128)
	for _, t := range qr.Rows {
		databases = append(databases, t[0].String())
	}
	return databases, nil
}

func (d *Dumper) filterDatabases(conn *Connection, filter *regexp.Regexp, invert bool) ([]string, error) {
	qr, err := conn.Fetch("SHOW DATABASES")
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0, 128)
	for _, t := range qr.Rows {
		if (!invert && filter.MatchString(t[0].String())) || (invert && !filter.MatchString(t[0].String())) {
			databases = append(databases, t[0].String())
		}
	}
	return databases, nil
}

// dumpableFieldNames returns a slice that contains valid field names for the dump.
func (d *Dumper) dumpableFieldNames(conn *Connection, table string) ([]string, error) {
	qr, err := conn.Fetch(fmt.Sprintf("SHOW FIELDS FROM `%s`", table))
	if err != nil {
		return nil, err
	}

	fields := make([]string, 0, len(qr.Rows))
	for _, t := range qr.Rows {
		if len(t) != 6 {
			return nil, fmt.Errorf("error fetching fields, expecting to have 6 columns, have: %d", len(t))
		}

		name := t[0].String()
		extra := t[5].String()

		// Can be either "VIRTUAL GENERATED" or "STORED GENERATED"
		// https://dev.mysql.com/doc/refman/8.0/en/show-columns.html
		if strings.Contains(extra, "VIRTUAL GENERATED") || strings.Contains(extra, "STORED GENERATED") {
			// Skip generated columns
			continue
		} else {
			fields = append(fields, name)
		}
	}

	return fields, nil
}

// writeFile used to write datas to file.
func writeFile(file string, data string) error {
	flag := os.O_RDWR | os.O_TRUNC
	if _, err := os.Stat(file); os.IsNotExist(err) {
		flag |= os.O_CREATE
	}
	f, err := os.OpenFile(file, flag, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := f.Write([]byte(data))
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

// escapeBytes used to escape the literal byte.
// See https://dev.mysql.com/doc/refman/5.7/en/string-literals.html
// for more information on how to escape string literals in MySQL.
func escapeBytes(data []byte) []byte {
	var buf bytes.Buffer
	for _, b := range data {
		switch b {
		case 0:
			buf.WriteString(`\0`)
		case '\'':
			buf.WriteString(`\'`)
		case '"':
			buf.WriteString(`\"`)
		case '\b':
			buf.WriteString(`\b`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		case 0x1A:
			buf.WriteString(`\Z`)
		case '\\':
			buf.WriteString(`\\`)
		default:
			buf.WriteByte(b)
		}
	}
	return buf.Bytes()
}
