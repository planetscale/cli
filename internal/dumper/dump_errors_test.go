package dumper

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/xelabs/go-mysqlstack/driver"
	querypb "github.com/xelabs/go-mysqlstack/sqlparser/depends/query"
	"github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"
	"github.com/xelabs/go-mysqlstack/xlog"

	qt "github.com/frankban/quicktest"
)

type errCloseRows struct {
	row          []sqltypes.Value
	rowSent      bool
	closeErr     error
	rowValuesErr error
	closeCalled  bool
}

func (r *errCloseRows) Next() bool {
	if r.rowSent {
		return false
	}
	r.rowSent = true
	return true
}

func (r *errCloseRows) Close() error {
	r.closeCalled = true
	return r.closeErr
}

func (r *errCloseRows) Datas() []byte { return nil }

func (r *errCloseRows) Bytes() int { return 0 }

func (r *errCloseRows) RowsAffected() uint64 { return 0 }

func (r *errCloseRows) LastInsertID() uint64 { return 0 }

func (r *errCloseRows) LastError() error { return nil }

func (r *errCloseRows) Fields() []*querypb.Field { return nil }

func (r *errCloseRows) RowValues() ([]sqltypes.Value, error) {
	if r.rowValuesErr != nil {
		return nil, r.rowValuesErr
	}
	return r.row, nil
}

type fakeConn struct {
	fieldsResult *sqltypes.Result
	streamRows   driver.Rows
}

func (f *fakeConn) Ping() error { return nil }

func (f *fakeConn) Quit() {}

func (f *fakeConn) Close() error { return nil }

func (f *fakeConn) Closed() bool { return false }

func (f *fakeConn) Cleanup() {}

func (f *fakeConn) NextPacket() ([]byte, error) { return nil, nil }

func (f *fakeConn) ConnectionID() uint32 { return 0 }

func (f *fakeConn) InitDB(string) error { return nil }

func (f *fakeConn) Command(byte) error { return nil }

func (f *fakeConn) Query(string) (driver.Rows, error) {
	return f.streamRows, nil
}

func (f *fakeConn) Exec(string) error { return nil }

func (f *fakeConn) FetchAll(string, int) (*sqltypes.Result, error) {
	return f.fieldsResult, nil
}

func (f *fakeConn) FetchAllWithFunc(string, int, driver.Func) (*sqltypes.Result, error) {
	return f.fieldsResult, nil
}

func (f *fakeConn) ComStatementPrepare(string) (*driver.Statement, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestDumpTableReportsCursorCloseError(t *testing.T) {
	c := qt.New(t)

	closeErr := errors.New("unexpected EOF")
	fieldsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{Name: "Field", Type: querypb.Type_VARCHAR},
			{Name: "Type", Type: querypb.Type_VARCHAR},
			{Name: "Null", Type: querypb.Type_VARCHAR},
			{Name: "Key", Type: querypb.Type_VARCHAR},
			{Name: "Default", Type: querypb.Type_VARCHAR},
			{Name: "Extra", Type: querypb.Type_VARCHAR},
		},
		Rows: [][]sqltypes.Value{
			testRow("id", ""),
		},
	}

	conn := &Connection{
		ID: 0,
		client: &fakeConn{
			fieldsResult: fieldsResult,
			streamRows: &errCloseRows{
				row: []sqltypes.Value{
					sqltypes.MakeTrusted(querypb.Type_INT32, []byte("1")),
				},
				closeErr: closeErr,
			},
		},
	}

	cfg := NewDefaultConfig()
	cfg.Outdir = c.TempDir()
	cfg.ChunksizeInMB = 128
	cfg.StmtSize = 1000000

	d := &Dumper{
		cfg: cfg,
		log: cmdutil.NewZapLogger(false),
	}

	err := d.dumpTable(context.Background(), conn, "test", "t1")
	c.Assert(err, qt.ErrorIs, closeErr)
}

func TestDumpTableClosesCursorOnRowValuesError(t *testing.T) {
	c := qt.New(t)

	rowErr := errors.New("row decode failed")
	fieldsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{Name: "Field", Type: querypb.Type_VARCHAR},
			{Name: "Type", Type: querypb.Type_VARCHAR},
			{Name: "Null", Type: querypb.Type_VARCHAR},
			{Name: "Key", Type: querypb.Type_VARCHAR},
			{Name: "Default", Type: querypb.Type_VARCHAR},
			{Name: "Extra", Type: querypb.Type_VARCHAR},
		},
		Rows: [][]sqltypes.Value{
			testRow("id", ""),
		},
	}

	rows := &errCloseRows{
		rowValuesErr: rowErr,
		closeErr:     errors.New("should not mask row error"),
	}

	conn := &Connection{
		ID: 0,
		client: &fakeConn{
			fieldsResult: fieldsResult,
			streamRows:   rows,
		},
	}

	cfg := NewDefaultConfig()
	cfg.Outdir = c.TempDir()
	cfg.ChunksizeInMB = 128
	cfg.StmtSize = 1000000

	d := &Dumper{
		cfg: cfg,
		log: cmdutil.NewZapLogger(false),
	}

	err := d.dumpTable(context.Background(), conn, "test", "t1")
	c.Assert(err, qt.ErrorIs, rowErr)
	c.Assert(rows.closeCalled, qt.IsTrue)
}

func TestRunReturnsDumpError(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	schemaResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{Name: "Table", Type: querypb.Type_VARCHAR},
			{Name: "Create Table", Type: querypb.Type_VARCHAR},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR,
					[]byte("CREATE TABLE `t1` (`id` int) ENGINE=InnoDB")),
			},
		},
	}

	tablesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{Name: "Tables_in_test", Type: querypb.Type_VARCHAR},
		},
		Rows: [][]sqltypes.Value{
			{sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1"))},
		},
	}

	viewsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{Name: "TABLE_NAME", Type: querypb.Type_VARCHAR},
		},
		Rows: [][]sqltypes.Value{},
	}

	fieldsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{Name: "Field", Type: querypb.Type_VARCHAR},
			{Name: "Type", Type: querypb.Type_VARCHAR},
			{Name: "Null", Type: querypb.Type_VARCHAR},
			{Name: "Key", Type: querypb.Type_VARCHAR},
			{Name: "Default", Type: querypb.Type_VARCHAR},
			{Name: "Extra", Type: querypb.Type_VARCHAR},
		},
		Rows: [][]sqltypes.Value{
			testRow("id", ""),
		},
	}

	dumpErr := errors.New("closed network connection")
	fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
	fakedbs.AddQueryPattern("show create table .*", schemaResult)
	fakedbs.AddQueryPattern("show tables from .*", tablesResult)
	fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
	fakedbs.AddQueryPattern("show fields from .*", fieldsResult)
	fakedbs.AddQueryErrorPattern("select .* from `test`\\.`t1` .*", dumpErr)
	fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})

	cfg := &Config{
		Database:      "test",
		Table:         "t1",
		Outdir:        c.TempDir(),
		User:          "mock",
		Password:      "mock",
		Address:       address,
		ChunksizeInMB: 1,
		Threads:       1,
		StmtSize:      10000,
		IntervalMs:    500,
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run(context.Background())
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "closed network connection")
}
