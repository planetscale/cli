package dumper

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xelabs/go-mysqlstack/sqlparser/depends/common"
	querypb "github.com/xelabs/go-mysqlstack/sqlparser/depends/query"
	"github.com/xelabs/go-mysqlstack/xlog"
)

// Config describes the settings to dump from a database.
type Config struct {
	User                 string
	Password             string
	Address              string
	ToUser               string
	ToPassword           string
	ToAddress            string
	ToDatabase           string
	ToEngine             string
	Database             string
	DatabaseRegexp       string
	DatabaseInvertRegexp bool
	Table                string
	Outdir               string
	SessionVars          string
	Format               string
	Threads              int
	ChunksizeInMB        int
	StmtSize             int
	Allbytes             uint64
	Allrows              uint64
	OverwriteTables      bool
	Wheres               map[string]string
	Selects              map[string]map[string]string
	Filters              map[string]map[string]string

	// Interval in millisecond.
	IntervalMs int
}

func NewDefaultConfig() *Config {
	return &Config{
		Format:        "mysql",
		Threads:       16,
		StmtSize:      1000000,
		IntervalMs:    10 * 1000,
		ChunksizeInMB: 128,
		SessionVars:   "set workload=olap;",
	}
}

type Dumper struct {
	cfg *Config
	log *xlog.Log
}

func NewDumper(cfg *Config) (*Dumper, error) {
	return &Dumper{
		cfg: cfg,
		log: xlog.NewStdLog(xlog.Level(xlog.INFO)),
	}, nil
}

func (d *Dumper) Run(ctx context.Context) error {
	initPool, err := NewPool(d.log, d.cfg.Threads, d.cfg.Address, d.cfg.User, d.cfg.Password, "", "")
	if err != nil {
		return err
	}
	defer initPool.Close()

	// Meta data.
	err = writeMetaData(d.cfg)
	if err != nil {
		return err
	}

	// database.
	conn := initPool.Get()
	var databases []string
	t := time.Now()
	if d.cfg.DatabaseRegexp != "" {
		r := regexp.MustCompile(d.cfg.DatabaseRegexp)
		databases, err = filterDatabases(d.log, conn, r, d.cfg.DatabaseInvertRegexp)
		if err != nil {
			return err
		}
	} else {
		if d.cfg.Database != "" {
			databases = strings.Split(d.cfg.Database, ",")
		} else {
			databases, err = allDatabases(d.log, conn)
			if err != nil {
				return err
			}
		}
	}
	for _, database := range databases {
		if err := dumpDatabaseSchema(d.log, conn, d.cfg, database); err != nil {
			return err
		}
	}

	tables := make([][]string, len(databases))
	for i, database := range databases {
		if d.cfg.Table != "" {
			tables[i] = strings.Split(d.cfg.Table, ",")
		} else {
			tables[i], err = allTables(d.log, conn, database)
			if err != nil {
				return err
			}
		}
	}
	initPool.Put(conn)

	// TODO(fatih): use errgroup
	var wg sync.WaitGroup
	for i, database := range databases {
		pool, err := NewPool(d.log, d.cfg.Threads/len(databases), d.cfg.Address, d.cfg.User, d.cfg.Password, d.cfg.SessionVars, database)
		if err != nil {
			return err
		}

		defer pool.Close()
		for _, table := range tables[i] {
			conn := initPool.Get()
			err := dumpTableSchema(d.log, conn, d.cfg, database, table)
			if err != nil {
				return err
			}

			initPool.Put(conn)

			conn = pool.Get()
			wg.Add(1)
			go func(conn *Connection, database string, table string) {
				defer func() {
					wg.Done()
					pool.Put(conn)
				}()

				d.log.Info("dumping.table[%s.%s].datas.thread[%d]...", database, table, conn.ID)
				if d.cfg.Format == "mysql" {
					err := dumpTable(d.log, conn, d.cfg, database, table)
					if err != nil {
						d.log.Error("error dumping table: %s", err)
					}
				} else if d.cfg.Format == "tsv" {
					err := dumpTableCsv(d.log, conn, d.cfg, database, table, '\t')
					if err != nil {
						d.log.Error("error dumping table in TSV: %s", err)
					}
				} else if d.cfg.Format == "csv" {
					err := dumpTableCsv(d.log, conn, d.cfg, database, table, ',')
					if err != nil {
						d.log.Error("error dumping table in CSV: %s", err)
					}
				} else {
					d.log.Error("error dumping table, unknown dump format: %s", d.cfg.Format)
				}

				d.log.Info("dumping.table[%s.%s].datas.thread[%d].done...", database, table, conn.ID)
			}(conn, database, table)
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
			d.log.Info("dumping.allbytes[%vMB].allrows[%v].time[%.2fsec].rates[%.2fMB/sec]...", allbytesMB, allrows, diff, rates)
		}
	}()

	wg.Wait()
	elapsed := time.Since(t).Seconds()
	d.log.Info("dumping.all.done.cost[%.2fsec].allrows[%v].allbytes[%v].rate[%.2fMB/s]", elapsed, d.cfg.Allrows, d.cfg.Allbytes, (float64(d.cfg.Allbytes/1024/1024) / elapsed))
	return nil
}

func writeMetaData(cfg *Config) error {
	file := fmt.Sprintf("%s/metadata", cfg.Outdir)
	return writeFile(file, "")
}

func dumpDatabaseSchema(log *xlog.Log, conn *Connection, cfg *Config, database string) error {
	schema := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", database)
	file := fmt.Sprintf("%s/%s-schema-create.sql", cfg.Outdir, database)
	err := writeFile(file, schema)
	if err != nil {
		return err
	}

	log.Info("dumping.database[%s].schema...", database)
	return nil
}

func dumpTableSchema(log *xlog.Log, conn *Connection, cfg *Config, database string, table string) error {
	qr, err := conn.Fetch(fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", database, table))
	if err != nil {
		return err
	}

	schema := qr.Rows[0][1].String() + ";\n"

	file := fmt.Sprintf("%s/%s.%s-schema.sql", cfg.Outdir, database, table)
	err = writeFile(file, schema)
	if err != nil {
		return err
	}

	log.Info("dumping.table[%s.%s].schema...", database, table)
	return nil
}

// Dump a table in "MySQL" (multi-inserts) format
func dumpTable(log *xlog.Log, conn *Connection, cfg *Config, database string, table string) error {
	var allBytes uint64
	var allRows uint64
	var where string
	var selfields []string

	fields := make([]string, 0, 16)
	{
		cursor, err := conn.StreamFetch(fmt.Sprintf("SELECT * FROM `%s`.`%s` LIMIT 1", database, table))
		if err != nil {
			return err
		}

		flds := cursor.Fields()
		for _, fld := range flds {
			log.Debug("dump -- %#v, %s, %s", cfg.Filters, table, fld.Name)
			if _, ok := cfg.Filters[table][fld.Name]; ok {
				continue
			}

			fields = append(fields, fmt.Sprintf("`%s`", fld.Name))
			replacement, ok := cfg.Selects[table][fld.Name]
			if ok {
				selfields = append(selfields, fmt.Sprintf("%s AS `%s`", replacement, fld.Name))
			} else {
				selfields = append(selfields, fmt.Sprintf("`%s`", fld.Name))
			}
		}
		err = cursor.Close()
		if err != nil {
			return err
		}
	}

	if v, ok := cfg.Wheres[table]; ok {
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
		atomic.AddUint64(&cfg.Allbytes, uint64(len(r)))
		atomic.AddUint64(&cfg.Allrows, 1)

		if stmtsize >= cfg.StmtSize {
			insertone := fmt.Sprintf("INSERT INTO `%s`(%s) VALUES\n%s", table, strings.Join(fields, ","), strings.Join(rows, ",\n"))
			inserts = append(inserts, insertone)
			rows = rows[:0]
			stmtsize = 0
		}

		if (chunkbytes / 1024 / 1024) >= cfg.ChunksizeInMB {
			query := strings.Join(inserts, ";\n") + ";\n"
			file := fmt.Sprintf("%s/%s.%s.%05d.sql", cfg.Outdir, database, table, fileNo)
			err = writeFile(file, query)
			if err != nil {
				return err
			}

			log.Info("dumping.table[%s.%s].rows[%v].bytes[%vMB].part[%v].thread[%d]", database, table, allRows, (allBytes / 1024 / 1024), fileNo, conn.ID)
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
		file := fmt.Sprintf("%s/%s.%s.%05d.sql", cfg.Outdir, database, table, fileNo)
		err = writeFile(file, query)
		if err != nil {
			return err
		}
	}
	err = cursor.Close()
	if err != nil {
		return err
	}

	log.Info("dumping.table[%s.%s].done.allrows[%v].allbytes[%vMB].thread[%d]...", database, table, allRows, (allBytes / 1024 / 1024), conn.ID)
	return nil
}

// Dump a table in CSV/TSV format
func dumpTableCsv(log *xlog.Log, conn *Connection, cfg *Config, database, table string, separator rune) error {
	var allBytes uint64
	var allRows uint64
	var where string
	var selfields []string
	var headerfields []string

	{
		cursor, err := conn.StreamFetch(fmt.Sprintf("SELECT * FROM `%s`.`%s` LIMIT 1", database, table))
		if err != nil {
			return err
		}

		flds := cursor.Fields()
		for _, fld := range flds {
			log.Debug("dump -- %#v, %s, %s", cfg.Filters, table, fld.Name)
			if _, ok := cfg.Filters[table][fld.Name]; ok {
				continue
			}

			headerfields = append(headerfields, fld.Name)
			replacement, ok := cfg.Selects[table][fld.Name]
			if ok {
				selfields = append(selfields, fmt.Sprintf("%s AS `%s`", replacement, fld.Name))
			} else {
				selfields = append(selfields, fmt.Sprintf("`%s`", fld.Name))
			}
		}
		err = cursor.Close()
		if err != nil {
			return err
		}

	}

	if v, ok := cfg.Wheres[table]; ok {
		where = fmt.Sprintf(" WHERE %v", v)
	}

	cursor, err := conn.StreamFetch(fmt.Sprintf("SELECT %s FROM `%s`.`%s` %s", strings.Join(selfields, ", "), database, table, where))
	if err != nil {
		return err
	}

	fileNo := 1
	file, err := os.Create(fmt.Sprintf("%s/%s.%s.%05d.csv", cfg.Outdir, database, table, fileNo))
	if err != nil {
		return err
	}

	writer := csv.NewWriter(file)
	writer.Comma = separator
	err = writer.Write(headerfields)
	if err != nil {
		return err
	}

	chunkbytes := 0

	inserts := make([]string, 0, 256)
	for cursor.Next() {
		row, err := cursor.RowValues()
		if err != nil {
			return err
		}

		values := make([]string, 0, 16)
		rowsize := 0
		for _, v := range row {
			if v.Raw() == nil {
				values = append(values, "NULL")
				rowsize += 4
			} else {
				str := v.String()
				switch {
				case v.IsSigned(), v.IsUnsigned(), v.IsFloat(), v.IsIntegral(), v.Type() == querypb.Type_DECIMAL:
					values = append(values, str)
					rowsize += len(str)
				default:
					values = append(values, fmt.Sprintf("%s", escapeBytes(v.Raw())))
					rowsize += len(v.Raw())
				}
			}
		}
		err = writer.Write(values)
		if err != nil {
			return err
		}
		chunkbytes += rowsize

		allRows++
		atomic.AddUint64(&cfg.Allbytes, uint64(rowsize))
		atomic.AddUint64(&cfg.Allrows, 1)

		if (chunkbytes / 1024 / 1024) >= cfg.ChunksizeInMB {
			writer.Flush()
			file, err := os.Create(fmt.Sprintf("%s/%s.%s.%05d.csv", cfg.Outdir, database, table, fileNo))
			if err != nil {
				return err
			}

			writer = csv.NewWriter(file)
			writer.Comma = separator
			err = writer.Write(headerfields)
			if err != nil {
				return err
			}

			log.Info("dumping.table[%s.%s].rows[%v].bytes[%vMB].part[%v].thread[%d]", database, table, allRows, (allBytes / 1024 / 1024), fileNo, conn.ID)
			inserts = inserts[:0]
			chunkbytes = 0
			fileNo++
		}
	}
	writer.Flush()
	err = cursor.Close()
	if err != nil {
		return err
	}

	log.Info("dumping.table[%s.%s].done.allrows[%v].allbytes[%vMB].thread[%d]...", database, table, allRows, (allBytes / 1024 / 1024), conn.ID)
	return nil
}

func allTables(log *xlog.Log, conn *Connection, database string) ([]string, error) {
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

func allDatabases(log *xlog.Log, conn *Connection) ([]string, error) {
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

func filterDatabases(log *xlog.Log, conn *Connection, filter *regexp.Regexp, invert bool) ([]string, error) {
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

	n, err := f.Write(common.StringToBytes(data))
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

// escapeBytes used to escape the literal byte.
func escapeBytes(bytes []byte) []byte {
	buffer := common.NewBuffer(128)
	for _, b := range bytes {
		// See https://dev.mysql.com/doc/refman/5.7/en/string-literals.html
		// for more information on how to escape string literals in MySQL.
		switch b {
		case 0:
			buffer.WriteString(`\0`)
		case '\'':
			buffer.WriteString(`\'`)
		case '"':
			buffer.WriteString(`\"`)
		case '\b':
			buffer.WriteString(`\b`)
		case '\n':
			buffer.WriteString(`\n`)
		case '\r':
			buffer.WriteString(`\r`)
		case '\t':
			buffer.WriteString(`\t`)
		case 0x1A:
			buffer.WriteString(`\Z`)
		case '\\':
			buffer.WriteString(`\\`)
		default:
			buffer.WriteU8(b)
		}
	}
	return buffer.Datas()
}
