package dumper

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/xelabs/go-mysqlstack/driver"
	querypb "github.com/xelabs/go-mysqlstack/sqlparser/depends/query"
	"github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"
	"github.com/xelabs/go-mysqlstack/xlog"

	qt "github.com/frankban/quicktest"
)

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

	// fakedbs.
	{
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select .*", selectResult)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Database:      "test",
		Outdir:        c.TempDir(),
		Format:        "mysql",
		User:          "mock",
		Password:      "mock",
		Address:       address,
		ChunksizeInMB: 1,
		Threads:       16,
		StmtSize:      10000,
		IntervalMs:    500,
		SessionVars:   "SET @@radon_streaming_fetch='ON', @@xx=1",
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run()
	c.Assert(err, qt.IsNil)

	dat, err := ioutil.ReadFile(cfg.Outdir + "/test.t1-05-11.00001.sql")
	c.Assert(err, qt.IsNil)

	want := strings.Contains(string(dat), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want, qt.IsTrue)
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

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select .* from `test1`.*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2`.*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Format:        "mysql",
		Outdir:        c.TempDir(),
		User:          "mock",
		Password:      "mock",
		Address:       address,
		ChunksizeInMB: 1,
		Threads:       16,
		StmtSize:      10000,
		IntervalMs:    500,
		SessionVars:   "SET @@radon_streaming_fetch='ON', @@xx=1",
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run()
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := ioutil.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := ioutil.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
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

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select .* from `test1`.*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2`.*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Format:        "mysql",
		Database:      "test1,test2",
		Outdir:        c.TempDir(),
		User:          "mock",
		Password:      "mock",
		Address:       address,
		ChunksizeInMB: 1,
		Threads:       16,
		StmtSize:      10000,
		IntervalMs:    500,
		SessionVars:   "SET @@radon_streaming_fetch='ON', @@xx=1",
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run()
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := ioutil.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := ioutil.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
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

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select .* from `test1`.*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2`.*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Format:         "mysql",
		DatabaseRegexp: "(test1|test2)",
		Outdir:         c.TempDir(),
		User:           "mock",
		Password:       "mock",
		Address:        address,
		ChunksizeInMB:  1,
		Threads:        16,
		StmtSize:       10000,
		IntervalMs:     500,
		SessionVars:    "SET @@radon_streaming_fetch='ON', @@xx=1",
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run()
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := ioutil.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := ioutil.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
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

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select .* from `test1`.*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2`.*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Format:         "mysql",
		DatabaseRegexp: "^[ets]+?[0-2]$",
		Outdir:         c.TempDir(),
		User:           "mock",
		Password:       "mock",
		Address:        address,
		ChunksizeInMB:  1,
		Threads:        16,
		StmtSize:       10000,
		IntervalMs:     500,
		SessionVars:    "SET @@radon_streaming_fetch='ON', @@xx=1",
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run()
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := ioutil.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := ioutil.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
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

	// fakedbs.
	{
		fakedbs.AddQueryPattern("show databases", databasesResult)
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("show create table .*", schemaResult)
		fakedbs.AddQueryPattern("show tables from .*", tablesResult)
		fakedbs.AddQueryPattern("select .* from `test1`.*", selectResult1)
		fakedbs.AddQueryPattern("select .* from `test2`.*", selectResult2)
		fakedbs.AddQueryPattern("set .*", &sqltypes.Result{})
	}

	cfg := &Config{
		Format:               "mysql",
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
		SessionVars:          "SET @@radon_streaming_fetch='ON', @@xx=1",
	}

	d, err := NewDumper(cfg)
	c.Assert(err, qt.IsNil)

	err = d.Run()
	c.Assert(err, qt.IsNil)

	dat_test1, err_test1 := ioutil.ReadFile(cfg.Outdir + "/test1.t1-05-11.00001.sql")
	c.Assert(err_test1, qt.IsNil)

	want_test1 := strings.Contains(string(dat_test1), `(11,"11\"xx\"","",NULL,210.01,NULL)`)
	c.Assert(want_test1, qt.IsTrue)

	dat_test2, err_test2 := ioutil.ReadFile(cfg.Outdir + "/test2.t1-05-11.00001.sql")
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
