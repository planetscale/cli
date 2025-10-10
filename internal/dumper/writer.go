package dumper

import "github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"

type TableWriter interface {
	Initialize(fieldNames []string) error
	WriteRow(row []sqltypes.Value) (bytesAdded int, err error)
	ShouldFlush() bool
	Flush(outdir, database, table string, fileNo int) error
	Close(outdir, database, table string, fileNo int) error
}
