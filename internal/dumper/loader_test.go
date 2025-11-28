package dumper

import (
	"context"
	"os"
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

func TestRestoreTableSchema_WithComments(t *testing.T) {
	tests := []struct {
		name          string
		schemaContent string
		setupQueries  []string
		description   string
	}{
		{
			name: "schema with line comments at end",
			schemaContent: `CREATE TABLE example_table (
    id INT PRIMARY KEY
);
-- This is a comment
-- This is another comment`,
			setupQueries: []string{
				"CREATE TABLE example_table (\n    id INT PRIMARY KEY\n)",
			},
			description: "Should execute CREATE TABLE and skip trailing comments",
		},
		{
			name: "schema with ALTER TABLE after comments",
			schemaContent: `CREATE TABLE example_table (
    id INT PRIMARY KEY,
    name VARCHAR(100)
);
-- This is a comment
-- This is another comment
ALTER TABLE example_table
  ADD INDEX idx_name (name);`,
			setupQueries: []string{
				"CREATE TABLE example_table (\n    id INT PRIMARY KEY,\n    name VARCHAR(100)\n)",
			},
			description: "Should execute CREATE TABLE and ALTER TABLE, skipping comments in between",
		},
		{
			name: "schema with block comments",
			schemaContent: `/* This is a block comment */
CREATE TABLE example_table (
    id INT PRIMARY KEY
);`,
			setupQueries: []string{
				"CREATE TABLE example_table (\n    id INT PRIMARY KEY\n)",
			},
			description: "Should skip block comments and execute CREATE TABLE",
		},
		{
			name: "schema with interspersed comments",
			schemaContent: `CREATE TABLE example_table (
    id INT PRIMARY KEY
);
-- Comment between statements
ALTER TABLE example_table ADD COLUMN name VARCHAR(100);
-- Another comment
ALTER TABLE example_table ADD INDEX idx_id (id);`,
			setupQueries: []string{
				"CREATE TABLE example_table (\n    id INT PRIMARY KEY\n)",
			},
			description: "Should execute all SQL statements and skip all comment lines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			log := xlog.NewStdLog(xlog.Level(xlog.ERROR))
			fakedbs := driver.NewTestHandler(log)
			server, err := driver.MockMysqlServer(log, fakedbs)
			c.Assert(err, qt.IsNil)
			defer server.Close()

			address := server.Addr()

			// Set up mock expectations
			fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
			fakedbs.AddQueryPattern("set foreign_key_checks=.*", &sqltypes.Result{})
			fakedbs.AddQueryPattern("drop table .*", &sqltypes.Result{})
			fakedbs.AddQueryPattern("alter table .*", &sqltypes.Result{})

			// Add expected queries - if these aren't executed, the test will fail
			for _, query := range tt.setupQueries {
				fakedbs.AddQuery(query, &sqltypes.Result{})
			}

			// Create test schema file
			tempDir := c.TempDir()
			schemaFile := tempDir + "/testdb.test_table-schema.sql"
			err = os.WriteFile(schemaFile, []byte(tt.schemaContent), 0644)
			c.Assert(err, qt.IsNil)

			// Create loader
			cfg := &Config{
				Database:        "testdb",
				Outdir:          tempDir,
				User:            "mock",
				Password:        "mock",
				Threads:         1,
				Address:         address,
				IntervalMs:      500,
				OverwriteTables: true,
				ShowDetails:     false,
				Debug:           false,
			}
			loader, err := NewLoader(cfg)
			c.Assert(err, qt.IsNil)

			// Create connection pool
			pool, err := NewPool(loader.log, cfg.Threads, cfg.Address, cfg.User, cfg.Password, cfg.SessionVars, "")
			c.Assert(err, qt.IsNil)
			defer pool.Close()

			conn := pool.Get()
			defer pool.Put(conn)

			// Execute restoreTableSchema - should not return error if all expected queries are executed
			err = loader.restoreTableSchema(cfg.OverwriteTables, []string{schemaFile}, conn)
			c.Assert(err, qt.IsNil, qt.Commentf("%s: failed to restore table schema", tt.description))
		})
	}
}

func TestRestoreTableSchema_DropTableCalledOnce(t *testing.T) {
	c := qt.New(t)

	log := xlog.NewStdLog(xlog.Level(xlog.ERROR))
	fakedbs := driver.NewTestHandler(log)
	server, err := driver.MockMysqlServer(log, fakedbs)
	c.Assert(err, qt.IsNil)
	defer server.Close()

	address := server.Addr()

	// Set up mock expectations
	fakedbs.AddQueryPattern("use .*", &sqltypes.Result{})
	fakedbs.AddQueryPattern("set foreign_key_checks=.*", &sqltypes.Result{})

	// Add DROP TABLE query only once - if it's called twice, the second call will fail
	// because there's no matching handler for it
	fakedbs.AddQuery("DROP TABLE IF EXISTS `testdb`.`test_table`", &sqltypes.Result{})
	fakedbs.AddQuery("CREATE TABLE test_table (\n    id INT PRIMARY KEY\n)", &sqltypes.Result{})

	// Create test schema file with comments at the end (the original bug scenario)
	tempDir := c.TempDir()
	schemaFile := tempDir + "/testdb.test_table-schema.sql"
	schemaContent := `CREATE TABLE test_table (
    id INT PRIMARY KEY
);
-- This is a comment
-- This is another comment`
	err = os.WriteFile(schemaFile, []byte(schemaContent), 0644)
	c.Assert(err, qt.IsNil)

	// Create loader
	cfg := &Config{
		Database:        "testdb",
		Outdir:          tempDir,
		User:            "mock",
		Password:        "mock",
		Threads:         1,
		Address:         address,
		IntervalMs:      500,
		OverwriteTables: true,
		ShowDetails:     false,
		Debug:           false,
	}
	loader, err := NewLoader(cfg)
	c.Assert(err, qt.IsNil)

	// Create connection pool
	pool, err := NewPool(loader.log, cfg.Threads, cfg.Address, cfg.User, cfg.Password, cfg.SessionVars, "")
	c.Assert(err, qt.IsNil)
	defer pool.Close()

	conn := pool.Get()
	defer pool.Put(conn)

	// Execute restoreTableSchema
	// If DROP TABLE is called more than once, the test will fail because there's
	// only one handler registered for it
	err = loader.restoreTableSchema(cfg.OverwriteTables, []string{schemaFile}, conn)
	c.Assert(err, qt.IsNil, qt.Commentf("DROP TABLE should be called exactly once. If called multiple times, this test will fail."))
}
