package dumper

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/xelabs/go-mysqlstack/driver"
	"github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"
	"github.com/xelabs/go-mysqlstack/xlog"
)

func TestLoader(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.DEBUG))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	c.Cleanup(func() { server.Close() })

	address := server.Addr()

	// fakedbs.
	{
		fakedbs.AddQueryPattern("create database if not exists `test.?`", &sqltypes.Result{})
		fakedbs.AddQuery("create table `t1-05-11` (`a` int(11) default null,`b` varchar(100) default null) engine=innodb", &sqltypes.Result{})
		fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("insert into .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("drop table .*", &sqltypes.Result{})
		fakedbs.AddQueryPattern("set foreign_key_checks=.*", &sqltypes.Result{})
	}

	cfg := &Config{
		Outdir:          c.TempDir(),
		User:            "mock",
		Password:        "mock",
		Threads:         16,
		Address:         address,
		IntervalMs:      500,
		OverwriteTables: true,
	}
	// Loader.
	loader, err := NewLoader(cfg)
	c.Assert(err, qt.IsNil)

	err = loader.Run(context.Background())
	c.Assert(err, qt.IsNil)
}
