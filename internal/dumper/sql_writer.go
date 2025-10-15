package dumper

import (
	"fmt"
	"strings"

	querypb "github.com/xelabs/go-mysqlstack/sqlparser/depends/query"
	"github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"
)

type sqlWriter struct {
	cfg        *Config
	table      string
	fields     []string
	rows       []string
	inserts    []string
	stmtsize   int
	chunkbytes int
}

func newSQLWriter(cfg *Config, table string) *sqlWriter {
	return &sqlWriter{
		cfg:     cfg,
		table:   table,
		rows:    make([]string, 0, 256),
		inserts: make([]string, 0, 256),
	}
}

func (w *sqlWriter) Initialize(fieldNames []string) error {
	w.fields = make([]string, len(fieldNames))
	for i, name := range fieldNames {
		w.fields[i] = fmt.Sprintf("`%s`", name)
	}
	return nil
}

func (w *sqlWriter) WriteRow(row []sqltypes.Value) (int, error) {
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
	w.rows = append(w.rows, r)

	rowBytes := len(r)
	w.stmtsize += rowBytes
	w.chunkbytes += rowBytes

	if w.stmtsize >= w.cfg.StmtSize {
		insertone := fmt.Sprintf("INSERT INTO `%s`(%s) VALUES\n%s", w.table, strings.Join(w.fields, ","), strings.Join(w.rows, ",\n"))
		w.inserts = append(w.inserts, insertone)
		w.rows = w.rows[:0]
		w.stmtsize = 0
	}

	return rowBytes, nil
}

func (w *sqlWriter) ShouldFlush() bool {
	return (w.chunkbytes / 1024 / 1024) >= w.cfg.ChunksizeInMB
}

func (w *sqlWriter) Flush(outdir, database, table string, fileNo int) error {
	query := strings.Join(w.inserts, ";\n") + ";\n"
	file := fmt.Sprintf("%s/%s.%s.%05d.sql", outdir, database, table, fileNo)
	err := writeFile(file, query)
	if err != nil {
		return err
	}

	w.inserts = w.inserts[:0]
	w.chunkbytes = 0
	return nil
}

func (w *sqlWriter) Close(outdir, database, table string, fileNo int) error {
	if w.chunkbytes > 0 {
		if len(w.rows) > 0 {
			insertone := fmt.Sprintf("INSERT INTO `%s`(%s) VALUES\n%s", w.table, strings.Join(w.fields, ","), strings.Join(w.rows, ",\n"))
			w.inserts = append(w.inserts, insertone)
		}
		return w.Flush(outdir, database, table, fileNo)
	}
	return nil
}
