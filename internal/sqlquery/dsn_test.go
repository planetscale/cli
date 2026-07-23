package sqlquery

import (
	"testing"

	qt "github.com/frankban/quicktest"
	gomysql "github.com/go-sql-driver/mysql"
)

// Shard-targeted keyspace names contain characters ("/", "@", "-") that must
// survive the round trip through the driver DSN.
func TestMySQLDSNEscapesShardTargets(t *testing.T) {
	c := qt.New(t)

	for _, dbName := range []string{
		"@primary",
		"mykeyspace",
		"mykeyspace/-80",
		"mykeyspace/80-c0@replica",
		"mykeyspace/-80@primary",
	} {
		dsn := mysqlDSN("127.0.0.1:3306", Options{Keyspace: dbName})
		parsed, err := gomysql.ParseDSN(dsn)
		c.Assert(err, qt.IsNil, qt.Commentf("dbname %q", dbName))
		c.Assert(parsed.DBName, qt.Equals, dbName)
	}
}
