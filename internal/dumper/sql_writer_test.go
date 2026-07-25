package dumper

import (
	"testing"

	qt "github.com/frankban/quicktest"
	querypb "github.com/xelabs/go-mysqlstack/sqlparser/depends/query"
	"github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"
)

func TestSQLWriterVARBINARYHex(t *testing.T) {
	c := qt.New(t)

	cfg := &Config{
		HexBlob:  true,
		StmtSize: 1024,
	}

	writer := newSQLWriter(cfg, "test_table")
	err := writer.Initialize([]string{"id", "data"})
	c.Assert(err, qt.IsNil)

	row := []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_INT32, []byte("123")),
		sqltypes.MakeTrusted(querypb.Type_VARBINARY, []byte("0x68656c6c6f")),
	}

	_, err = writer.WriteRow(row)
	c.Assert(err, qt.IsNil)

	inserts := writer.inserts
	c.Assert(len(inserts), qt.Equals, 0)

	rows := writer.rows
	c.Assert(len(rows), qt.Equals, 1)

	expectedRow := "(123,0x68656c6c6f)"
	c.Assert(rows[0], qt.Equals, expectedRow)
}

func TestSQLWriterVARBINARYNotQuoted(t *testing.T) {
	c := qt.New(t)

	cfg := &Config{
		HexBlob:  true,
		StmtSize: 1024,
	}

	writer := newSQLWriter(cfg, "test_table")
	err := writer.Initialize([]string{"id", "data"})
	c.Assert(err, qt.IsNil)

	hexValue := sqltypes.MakeTrusted(querypb.Type_VARBINARY, []byte("0x776f726c64"))

	row := []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_INT32, []byte("456")),
		hexValue,
	}

	_, err = writer.WriteRow(row)
	c.Assert(err, qt.IsNil)

	rows := writer.rows
	c.Assert(len(rows), qt.Equals, 1)

	rowStr := rows[0]
	c.Assert(rowStr, qt.Not(qt.Contains), `"0x`)
	c.Assert(rowStr, qt.Contains, "0x776f726c64")
}

func TestSQLWriterVARBINARYQuotedWithoutHexBlob(t *testing.T) {
	c := qt.New(t)

	cfg := &Config{
		StmtSize: 1024,
	}

	writer := newSQLWriter(cfg, "test_table")
	err := writer.Initialize([]string{"id", "data"})
	c.Assert(err, qt.IsNil)

	row := []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_INT32, []byte("456")),
		sqltypes.MakeTrusted(querypb.Type_VARBINARY, []byte("world")),
	}

	_, err = writer.WriteRow(row)
	c.Assert(err, qt.IsNil)

	rows := writer.rows
	c.Assert(len(rows), qt.Equals, 1)

	rowStr := rows[0]
	c.Assert(rowStr, qt.Contains, `"world"`)
	c.Assert(rowStr, qt.Not(qt.Contains), "0x776f726c64")
}

func TestSQLWriterVARINTAR(t *testing.T) {
	c := qt.New(t)

	cfg := &Config{
		StmtSize: 1024,
	}

	writer := newSQLWriter(cfg, "test_table")
	err := writer.Initialize([]string{"id", "name"})
	c.Assert(err, qt.IsNil)

	row := []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_INT32, []byte("789")),
		sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("hello world")),
	}

	_, err = writer.WriteRow(row)
	c.Assert(err, qt.IsNil)

	rows := writer.rows
	c.Assert(len(rows), qt.Equals, 1)

	expectedRow := `(789,"hello world")`
	c.Assert(rows[0], qt.Equals, expectedRow)
}

func TestSQLWriterNullValue(t *testing.T) {
	c := qt.New(t)

	cfg := &Config{
		StmtSize: 1024,
	}

	writer := newSQLWriter(cfg, "test_table")
	err := writer.Initialize([]string{"id", "data"})
	c.Assert(err, qt.IsNil)

	row := []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_INT32, []byte("123")),
		sqltypes.NULL,
	}

	_, err = writer.WriteRow(row)
	c.Assert(err, qt.IsNil)

	rows := writer.rows
	c.Assert(len(rows), qt.Equals, 1)

	expectedRow := "(123,NULL)"
	c.Assert(rows[0], qt.Equals, expectedRow)
}

func TestSQLWriterNumericTypes(t *testing.T) {
	c := qt.New(t)

	cfg := &Config{
		StmtSize: 1024,
	}

	writer := newSQLWriter(cfg, "test_table")
	err := writer.Initialize([]string{"int_col", "decimal_col"})
	c.Assert(err, qt.IsNil)

	row := []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_INT32, []byte("42")),
		sqltypes.MakeTrusted(querypb.Type_DECIMAL, []byte("99.99")),
	}

	_, err = writer.WriteRow(row)
	c.Assert(err, qt.IsNil)

	rows := writer.rows
	c.Assert(len(rows), qt.Equals, 1)

	expectedRow := "(42,99.99)"
	c.Assert(rows[0], qt.Equals, expectedRow)
}
