package dumper

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/xelabs/go-mysqlstack/driver"
	querypb "github.com/xelabs/go-mysqlstack/sqlparser/depends/query"
	"github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"
	"github.com/xelabs/go-mysqlstack/xlog"

	qt "github.com/frankban/quicktest"
)

func testRow(name, extra string) []sqltypes.Value {
	return []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte(name)),
		sqltypes.MakeTrusted(querypb.Type_BLOB, []byte("varchar(255)")),
		sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("YES")),
		sqltypes.MakeTrusted(querypb.Type_BINARY, []byte("")),
		sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, []byte("NULL")),
		sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte(extra)),
	}
}

func TestDumper(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	selectResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
			{
				Name: "name",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "namei1",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "null",
				Type: querypb.Type_NULL_TYPE,
			},
			{
				Name: "decimal",
				Type: querypb.Type_DECIMAL,
			},
			{
				Name: "datetime",
				Type: querypb.Type_DATETIME,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("11")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("11\"xx\"")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("")),
			sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil),
			sqltypes.MakeTrusted(querypb.Type_DECIMAL, []byte("210.01")),
			sqltypes.NULL,
		}
		selectResult.Rows = append(selectResult.Rows, row)
	}

	schemaResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Table",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "Create Table",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR,
					[]byte("CREATE TABLE `t1-05-11` (`a` int(11) DEFAULT NULL,`b` varchar(100) DEFAULT NULL) ENGINE=InnoDB")),
			},
		},
	}

	tablesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Tables_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1-05-11")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t2-05-11")),
			},
		},
	}

	viewsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Views_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v1-2024-10-25")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v2-2024-10-25")),
			},
		},
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
			testRow("name", ""),
			testRow("namei1", ""),
			testRow("null", ""),
			testRow("decimal", ""),
			testRow("datetime", ""),
			testRow("not_deleted", "virtual generated"),
		},
	}

	// fakedbs.
	{
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("show fields from .*", fieldsResult)
		fakedbs.AddQueryPattern("select .* from `test`\\..* .*", selectResult)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Database:      "test",
		Outdir:        c.TempDir(),
		User:          "mock",
		Password:      "mock",
		Address:       address,
		ChunksizeInMB: 1,
		Threads:       16,
		StmtSize:      10000,
		IntervalMs:    500,
		SessionVars:   []string{"SET @@radon_streaming_fetch='ON', @@xx=1"},
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run(context.Background())
	c.Assert(err, qt.IsNil)

	dat, err := os.ReadFile(cfg.Outdir + "/test.t1-05-11.00001.sql")
	c.Assert(err, qt.IsNil)

	want := strings.Contains(string(dat), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want, qt.IsTrue)
}

func TestDumperUseUseReplica(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	selectResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
			{
				Name: "name",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "namei1",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "null",
				Type: querypb.Type_NULL_TYPE,
			},
			{
				Name: "decimal",
				Type: querypb.Type_DECIMAL,
			},
			{
				Name: "datetime",
				Type: querypb.Type_DATETIME,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("11")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("11\"xx\"")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("")),
			sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil),
			sqltypes.MakeTrusted(querypb.Type_DECIMAL, []byte("210.01")),
			sqltypes.NULL,
		}
		selectResult.Rows = append(selectResult.Rows, row)
	}

	schemaResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Table",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "Create Table",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR,
					[]byte("CREATE TABLE `t1-05-11` (`a` int(11) DEFAULT NULL,`b` varchar(100) DEFAULT NULL) ENGINE=InnoDB")),
			},
		},
	}

	tablesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Tables_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1-05-11")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t2-05-11")),
			},
		},
	}

	viewsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Views_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v1-2024-10-25")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v2-2024-10-25")),
			},
		},
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
			testRow("name", ""),
			testRow("namei1", ""),
			testRow("null", ""),
			testRow("decimal", ""),
			testRow("datetime", ""),
			testRow("not_deleted", "virtual generated"),
		},
	}

	// fakedbs.
	{
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("show fields from .*", fieldsResult)
		fakedbs.AddQueryPattern("select .* FROM `test@replica`\\..* .*", selectResult)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Database:      "test",
		Outdir:        c.TempDir(),
		User:          "mock",
		Password:      "mock",
		Address:       address,
		ChunksizeInMB: 1,
		Threads:       16,
		StmtSize:      10000,
		IntervalMs:    500,
		SessionVars:   []string{"SET @@radon_streaming_fetch='ON', @@xx=1"},
		UseReplica:    true,
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run(context.Background())
	c.Assert(err, qt.IsNil)

	dat, err := os.ReadFile(cfg.Outdir + "/test.t1-05-11.00001.sql")
	c.Assert(err, qt.IsNil)

	want := strings.Contains(string(dat), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want, qt.IsTrue)
}

func TestDumperGeneratedFields(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	selectResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
			{
				Name: "namei1",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "name",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "null",
				Type: querypb.Type_NULL_TYPE,
			},
			{
				Name: "decimal",
				Type: querypb.Type_DECIMAL,
			},
			{
				Name: "datetime",
				Type: querypb.Type_DATETIME,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("11")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("11\"xx\"")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("")),
			sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil),
			sqltypes.MakeTrusted(querypb.Type_DECIMAL, []byte("210.01")),
			sqltypes.NULL,
		}
		selectResult.Rows = append(selectResult.Rows, row)
	}

	schemaResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Table",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "Create Table",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR,
					[]byte("CREATE TABLE `t1-05-11` (`a` int(11) DEFAULT NULL,`b` varchar(100) DEFAULT NULL) ENGINE=InnoDB")),
			},
		},
	}

	tablesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Tables_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1-05-11")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t2-05-11")),
			},
		},
	}

	viewsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Views_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v1-2024-10-25")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v2-2024-10-25")),
			},
		},
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
			testRow("name", "VIRTUAL GENERATED"),
			testRow("namei1", ""),
			testRow("null", ""),
			testRow("decimal", ""),
			testRow("datetime", "DEFAULT_GENERATED"),
			testRow("not_deleted", "STORED GENERATED"),
		},
	}

	// fakedbs.
	{
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("show fields from .*", fieldsResult)
		fakedbs.AddQueryPattern("select .* from `test`\\..* .*", selectResult)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Database:      "test",
		Outdir:        c.TempDir(),
		User:          "mock",
		Password:      "mock",
		Address:       address,
		ChunksizeInMB: 1,
		Threads:       16,
		StmtSize:      10000,
		IntervalMs:    500,
		SessionVars:   []string{"SET @@radon_streaming_fetch='ON', @@xx=1"},
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run(context.Background())
	c.Assert(err, qt.IsNil)

	dat, err := os.ReadFile(cfg.Outdir + "/test.t1-05-11.00001.sql")
	c.Assert(err, qt.IsNil)

	insStmt := "INSERT INTO `t1-05-11`(`id`,`namei1`,`null`,`decimal`,`datetime`) VALUES"
	c.Assert(string(dat), qt.Contains, insStmt)
}

func TestDumperAll(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	selectResult1 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
			{
				Name: "name",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "namei1",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "null",
				Type: querypb.Type_NULL_TYPE,
			},
			{
				Name: "decimal",
				Type: querypb.Type_DECIMAL,
			},
			{
				Name: "datetime",
				Type: querypb.Type_DATETIME,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("11")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("11\"xx\"")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("")),
			sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil),
			sqltypes.MakeTrusted(querypb.Type_DECIMAL, []byte("210.01")),
			sqltypes.NULL,
		}
		selectResult1.Rows = append(selectResult1.Rows, row)
	}

	selectResult2 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("1337")),
		}
		selectResult2.Rows = append(selectResult2.Rows, row)
	}

	schemaResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Table",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "Create Table",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR,
					[]byte("CREATE TABLE `t1-05-11` (`a` int(11) DEFAULT NULL,`b` varchar(100) DEFAULT NULL) ENGINE=InnoDB")),
			},
		},
	}

	tablesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Tables_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1-05-11")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t2-05-11")),
			},
		},
	}

	viewsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Views_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v1-2024-10-25")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v2-2024-10-25")),
			},
		},
	}

	databasesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Databases_in_database",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test1")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test2")),
			},
		},
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
			testRow("name", ""),
			testRow("namei1", ""),
			testRow("null", ""),
			testRow("decimal", ""),
			testRow("datetime", ""),
			testRow("not_deleted", "virtual generated"),
		},
	}

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test1' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test2' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("show fields from .*", fieldsResult)
		fakedbs.AddQueryPattern("select .* from `test1`\\..* .*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2`\\..* .*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Outdir:        c.TempDir(),
		User:          "mock",
		Password:      "mock",
		Address:       address,
		ChunksizeInMB: 1,
		Threads:       16,
		StmtSize:      10000,
		IntervalMs:    500,
		SessionVars:   []string{"SET @@radon_streaming_fetch='ON', @@xx=1"},
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run(context.Background())
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := os.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := os.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
	c.Assert(err_test2, qt.IsNil)

	want_test2 := strings.Contains(string(dat_test2), `(1337)`)
	c.Assert(want_test2, qt.IsTrue)
}

func TestDumperAllUseReplica(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	selectResult1 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
			{
				Name: "name",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "namei1",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "null",
				Type: querypb.Type_NULL_TYPE,
			},
			{
				Name: "decimal",
				Type: querypb.Type_DECIMAL,
			},
			{
				Name: "datetime",
				Type: querypb.Type_DATETIME,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("11")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("11\"xx\"")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("")),
			sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil),
			sqltypes.MakeTrusted(querypb.Type_DECIMAL, []byte("210.01")),
			sqltypes.NULL,
		}
		selectResult1.Rows = append(selectResult1.Rows, row)
	}

	selectResult2 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("1337")),
		}
		selectResult2.Rows = append(selectResult2.Rows, row)
	}

	schemaResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Table",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "Create Table",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR,
					[]byte("CREATE TABLE `t1-05-11` (`a` int(11) DEFAULT NULL,`b` varchar(100) DEFAULT NULL) ENGINE=InnoDB")),
			},
		},
	}

	tablesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Tables_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1-05-11")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t2-05-11")),
			},
		},
	}

	viewsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Views_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v1-2024-10-25")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v2-2024-10-25")),
			},
		},
	}

	databasesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Databases_in_database",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test1")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test2")),
			},
		},
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
			testRow("name", ""),
			testRow("namei1", ""),
			testRow("null", ""),
			testRow("decimal", ""),
			testRow("datetime", ""),
			testRow("not_deleted", "virtual generated"),
		},
	}

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test1' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test2' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("show fields from .*", fieldsResult)
		fakedbs.AddQueryPattern("select .* from `test1@replica`\\..* .*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2@replica`\\..* .*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Outdir:        c.TempDir(),
		User:          "mock",
		Password:      "mock",
		Address:       address,
		ChunksizeInMB: 1,
		Threads:       16,
		StmtSize:      10000,
		IntervalMs:    500,
		SessionVars:   []string{"SET @@radon_streaming_fetch='ON', @@xx=1"},
		UseReplica:    true,
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run(context.Background())
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := os.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := os.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
	c.Assert(err_test2, qt.IsNil)

	want_test2 := strings.Contains(string(dat_test2), `(1337)`)
	c.Assert(want_test2, qt.IsTrue)
}

func TestDumperMultiple(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	selectResult1 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
			{
				Name: "name",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "namei1",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "null",
				Type: querypb.Type_NULL_TYPE,
			},
			{
				Name: "decimal",
				Type: querypb.Type_DECIMAL,
			},
			{
				Name: "datetime",
				Type: querypb.Type_DATETIME,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("11")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("11\"xx\"")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("")),
			sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil),
			sqltypes.MakeTrusted(querypb.Type_DECIMAL, []byte("210.01")),
			sqltypes.NULL,
		}
		selectResult1.Rows = append(selectResult1.Rows, row)
	}

	selectResult2 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("1337")),
		}
		selectResult2.Rows = append(selectResult2.Rows, row)
	}

	schemaResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Table",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "Create Table",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR,
					[]byte("CREATE TABLE `t1-05-11` (`a` int(11) DEFAULT NULL,`b` varchar(100) DEFAULT NULL) ENGINE=InnoDB")),
			},
		},
	}

	tablesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Tables_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1-05-11")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t2-05-11")),
			},
		},
	}

	viewsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Views_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v1-2024-10-25")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v2-2024-10-25")),
			},
		},
	}

	databasesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Databases_in_database",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test1")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test2")),
			},
		},
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
			testRow("name", ""),
			testRow("namei1", ""),
			testRow("null", ""),
			testRow("decimal", ""),
			testRow("datetime", ""),
			testRow("not_deleted", "virtual generated"),
		},
	}

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test1' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test2' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("show fields from .*", fieldsResult)
		fakedbs.AddQueryPattern("select .* from `test1`\\..* .*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2`\\..* .*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Database:      "test1,test2",
		Outdir:        c.TempDir(),
		User:          "mock",
		Password:      "mock",
		Address:       address,
		ChunksizeInMB: 1,
		Threads:       16,
		StmtSize:      10000,
		IntervalMs:    500,
		SessionVars:   []string{"SET @@radon_streaming_fetch='ON', @@xx=1"},
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run(context.Background())
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := os.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := os.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
	c.Assert(err_test2, qt.IsNil)

	want_test2 := strings.Contains(string(dat_test2), `(1337)`)
	c.Assert(want_test2, qt.IsTrue)
}

func TestDumperMultipleUseReplica(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	selectResult1 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
			{
				Name: "name",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "namei1",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "null",
				Type: querypb.Type_NULL_TYPE,
			},
			{
				Name: "decimal",
				Type: querypb.Type_DECIMAL,
			},
			{
				Name: "datetime",
				Type: querypb.Type_DATETIME,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("11")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("11\"xx\"")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("")),
			sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil),
			sqltypes.MakeTrusted(querypb.Type_DECIMAL, []byte("210.01")),
			sqltypes.NULL,
		}
		selectResult1.Rows = append(selectResult1.Rows, row)
	}

	selectResult2 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("1337")),
		}
		selectResult2.Rows = append(selectResult2.Rows, row)
	}

	schemaResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Table",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "Create Table",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR,
					[]byte("CREATE TABLE `t1-05-11` (`a` int(11) DEFAULT NULL,`b` varchar(100) DEFAULT NULL) ENGINE=InnoDB")),
			},
		},
	}

	tablesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Tables_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1-05-11")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t2-05-11")),
			},
		},
	}

	viewsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Views_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v1-2024-10-25")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v2-2024-10-25")),
			},
		},
	}

	databasesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Databases_in_database",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test1")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test2")),
			},
		},
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
			testRow("name", ""),
			testRow("namei1", ""),
			testRow("null", ""),
			testRow("decimal", ""),
			testRow("datetime", ""),
			testRow("not_deleted", "virtual generated"),
		},
	}

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test1' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test2' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("show fields from .*", fieldsResult)
		fakedbs.AddQueryPattern("select .* from `test1@replica`\\..* .*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2@replica`\\..* .*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Database:      "test1,test2",
		Outdir:        c.TempDir(),
		User:          "mock",
		Password:      "mock",
		Address:       address,
		ChunksizeInMB: 1,
		Threads:       16,
		StmtSize:      10000,
		IntervalMs:    500,
		SessionVars:   []string{"SET @@radon_streaming_fetch='ON', @@xx=1"},
		UseReplica:    true,
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run(context.Background())
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := os.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := os.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
	c.Assert(err_test2, qt.IsNil)

	want_test2 := strings.Contains(string(dat_test2), `(1337)`)
	c.Assert(want_test2, qt.IsTrue)
}

func TestDumperSimpleRegexp(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	selectResult1 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
			{
				Name: "name",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "namei1",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "null",
				Type: querypb.Type_NULL_TYPE,
			},
			{
				Name: "decimal",
				Type: querypb.Type_DECIMAL,
			},
			{
				Name: "datetime",
				Type: querypb.Type_DATETIME,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("11")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("11\"xx\"")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("")),
			sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil),
			sqltypes.MakeTrusted(querypb.Type_DECIMAL, []byte("210.01")),
			sqltypes.NULL,
		}
		selectResult1.Rows = append(selectResult1.Rows, row)
	}

	selectResult2 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("1337")),
		}
		selectResult2.Rows = append(selectResult2.Rows, row)
	}

	schemaResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Table",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "Create Table",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR,
					[]byte("CREATE TABLE `t1-05-11` (`a` int(11) DEFAULT NULL,`b` varchar(100) DEFAULT NULL) ENGINE=InnoDB")),
			},
		},
	}

	tablesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Tables_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1-05-11")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t2-05-11")),
			},
		},
	}

	viewsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Views_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v1-2024-10-25")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v2-2024-10-25")),
			},
		},
	}

	databasesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Databases_in_database",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test1")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test2")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test3")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test4")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test5")),
			},
		},
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
			testRow("name", ""),
			testRow("namei1", ""),
			testRow("null", ""),
			testRow("decimal", ""),
			testRow("datetime", ""),
			testRow("not_deleted", "virtual generated"),
		},
	}

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test1' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test2' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("show fields from .*", fieldsResult)
		fakedbs.AddQueryPattern("select .* from `test1`\\..* .*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2`\\..* .*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		DatabaseRegexp: "(test1|test2)",
		Outdir:         c.TempDir(),
		User:           "mock",
		Password:       "mock",
		Address:        address,
		ChunksizeInMB:  1,
		Threads:        16,
		StmtSize:       10000,
		IntervalMs:     500,
		SessionVars:    []string{"SET @@radon_streaming_fetch='ON', @@xx=1"},
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run(context.Background())
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := os.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := os.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
	c.Assert(err_test2, qt.IsNil)

	want_test2 := strings.Contains(string(dat_test2), `(1337)`)
	c.Assert(want_test2, qt.IsTrue)
}

func TestDumperComplexRegexp(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	selectResult1 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
			{
				Name: "name",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "namei1",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "null",
				Type: querypb.Type_NULL_TYPE,
			},
			{
				Name: "decimal",
				Type: querypb.Type_DECIMAL,
			},
			{
				Name: "datetime",
				Type: querypb.Type_DATETIME,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("11")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("11\"xx\"")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("")),
			sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil),
			sqltypes.MakeTrusted(querypb.Type_DECIMAL, []byte("210.01")),
			sqltypes.NULL,
		}
		selectResult1.Rows = append(selectResult1.Rows, row)
	}

	selectResult2 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("1337")),
		}
		selectResult2.Rows = append(selectResult2.Rows, row)
	}

	schemaResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Table",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "Create Table",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR,
					[]byte("CREATE TABLE `t1-05-11` (`a` int(11) DEFAULT NULL,`b` varchar(100) DEFAULT NULL) ENGINE=InnoDB")),
			},
		},
	}

	tablesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Tables_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1-05-11")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t2-05-11")),
			},
		},
	}

	viewsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Views_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v1-2024-10-25")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v2-2024-10-25")),
			},
		},
	}

	databasesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Databases_in_database",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test1")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test2")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("foo1")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("bar2")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test5")),
			},
		},
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
			testRow("name", ""),
			testRow("namei1", ""),
			testRow("null", ""),
			testRow("decimal", ""),
			testRow("datetime", ""),
			testRow("not_deleted", "virtual generated"),
		},
	}

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test1' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test2' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("show fields from .*", fieldsResult)
		fakedbs.AddQueryPattern("select .* from `test1`\\..* .*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2`\\..* .*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		DatabaseRegexp: "^[ets]+?[0-2]$",
		Outdir:         c.TempDir(),
		User:           "mock",
		Password:       "mock",
		Address:        address,
		ChunksizeInMB:  1,
		Threads:        16,
		StmtSize:       10000,
		IntervalMs:     500,
		SessionVars:    []string{"SET @@radon_streaming_fetch='ON', @@xx=1"},
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run(context.Background())
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := os.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := os.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
	c.Assert(err_test2, qt.IsNil)

	want_test2 := strings.Contains(string(dat_test2), `(1337)`)
	c.Assert(want_test2, qt.IsTrue)
}

func TestDumperInvertMatch(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	selectResult1 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
			{
				Name: "name",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "namei1",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "null",
				Type: querypb.Type_NULL_TYPE,
			},
			{
				Name: "decimal",
				Type: querypb.Type_DECIMAL,
			},
			{
				Name: "datetime",
				Type: querypb.Type_DATETIME,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("11")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("11\"xx\"")),
			sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("")),
			sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil),
			sqltypes.MakeTrusted(querypb.Type_DECIMAL, []byte("210.01")),
			sqltypes.NULL,
		}
		selectResult1.Rows = append(selectResult1.Rows, row)
	}

	selectResult2 := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "id",
				Type: querypb.Type_INT32,
			},
		},
		Rows: make([][]sqltypes.Value, 0, 256),
	}

	for i := 0; i < 201710; i++ {
		row := []sqltypes.Value{
			sqltypes.MakeTrusted(querypb.Type_INT32, []byte("1337")),
		}
		selectResult2.Rows = append(selectResult2.Rows, row)
	}

	schemaResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Table",
				Type: querypb.Type_VARCHAR,
			},
			{
				Name: "Create Table",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR,
					[]byte("CREATE TABLE `t1-05-11` (`a` int(11) DEFAULT NULL,`b` varchar(100) DEFAULT NULL) ENGINE=InnoDB")),
			},
		},
	}

	tablesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Tables_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t1-05-11")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("t2-05-11")),
			},
		},
	}

	viewsResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Views_in_test",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v1-2024-10-25")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("v2-2024-10-25")),
			},
		},
	}

	databasesResult := &sqltypes.Result{
		Fields: []*querypb.Field{
			{
				Name: "Databases_in_database",
				Type: querypb.Type_VARCHAR,
			},
		},
		Rows: [][]sqltypes.Value{
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test1")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("test2")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("mysql")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("sys")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("information_schema")),
			},
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("performance_schema")),
			},
		},
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
			{
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("not_deleted")),
				sqltypes.MakeTrusted(querypb.Type_BLOB, []byte("int")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("YES")),
				sqltypes.MakeTrusted(querypb.Type_BINARY, []byte("")),
				sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, []byte("NULL")),
				sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("VIRTUAL GENERATED")),
			},
		},
	}

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test1' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("select table_name \n\t\t\t from information_schema.tables \n\t\t\t where table_schema like 'test2' \n\t\t\t and table_type = 'view'\n\t\t\t", viewsResult)
		fakedbs.AddQueryPattern("show fields from .*", fieldsResult)
		fakedbs.AddQueryPattern("select .* from `test1`\\..* .*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2`\\..* .*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		DatabaseRegexp:       "^(mysql|sys|information_schema|performance_schema)$",
		DatabaseInvertRegexp: true,
		Outdir:               c.TempDir(),
		User:                 "mock",
		Password:             "mock",
		Address:              address,
		ChunksizeInMB:        1,
		Threads:              16,
		StmtSize:             10000,
		IntervalMs:           500,
		SessionVars:          []string{"SET @@radon_streaming_fetch='ON', @@xx=1"},
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run(context.Background())
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := os.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := os.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
	c.Assert(err_test2, qt.IsNil)

	want_test2 := strings.Contains(string(dat_test2), `(1337)`)
	c.Assert(want_test2, qt.IsTrue)
}

func TestWriteFile(t *testing.T) {
	c := qt.New(t)

	file := "/tmp/xx.txt"
	defer os.Remove(file)

	{
		err := writeFile(file, "fake")
		c.Assert(err, qt.IsNil)
	}

	{
		err := writeFile("/xxu01/xx.txt", "fake")
		c.Assert(err, qt.Not(qt.IsNil))
	}
}

func TestEscapeBytes(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		v   []byte
		exp []byte
	}{
		{[]byte("simple"), []byte("simple")},
		{[]byte(`simplers's "world"`), []byte(`simplers\'s \"world\"`)},
		{[]byte("\x00'\"\b\n\r"), []byte(`\0\'\"\b\n\r`)},
		{[]byte("\t\x1A\\"), []byte(`\t\Z\\`)},
	}
	for _, tt := range tests {
		got := escapeBytes(tt.v)
		want := tt.exp

		c.Assert(want, qt.DeepEquals, got)
	}
}
