package dumper

import (
	"testing"

	qt "github.com/frankban/quicktest"
	querypb "github.com/xelabs/go-mysqlstack/sqlparser/depends/query"
	"github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"
)

type mockTableWriter struct {
	initialized bool
	rows        [][]sqltypes.Value
	fieldNames  []string
}

func (m *mockTableWriter) Initialize(fieldNames []string) error {
	m.fieldNames = fieldNames
	m.initialized = true
	return nil
}

func (m *mockTableWriter) WriteRow(row []sqltypes.Value) (int, error) {
	m.rows = append(m.rows, row)
	return len(row), nil
}

func (m *mockTableWriter) ShouldFlush() bool {
	return false
}

func (m *mockTableWriter) Flush(outdir, database, table string, fileNo int) error {
	return nil
}

func (m *mockTableWriter) Close(outdir, database, table string, fileNo int) error {
	return nil
}

func TestHexBlobWrapperWriteRowBLOB(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"id", "data"})

	row := []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_INT32, []byte("123")),
		sqltypes.MakeTrusted(querypb.Type_BLOB, []byte("hello")),
	}

	_, err := wrapper.WriteRow(row)
	c.Assert(err, qt.IsNil)
	c.Assert(len(mock.rows), qt.Equals, 1)

	// Check that BLOB was hex-encoded
	encodedValue := mock.rows[0][1]
	c.Assert(string(encodedValue.Raw()), qt.Equals, "0x68656c6c6f")
	c.Assert(encodedValue.Type(), qt.Equals, querypb.Type_VARBINARY)

	// Check that non-BLOB value was unchanged
	c.Assert(mock.rows[0][0].String(), qt.Equals, row[0].String())
}

func TestHexBlobWrapperWriteRowVARBINARY(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"id", "data"})

	row := []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_INT32, []byte("456")),
		sqltypes.MakeTrusted(querypb.Type_VARBINARY, []byte("world")),
	}

	_, err := wrapper.WriteRow(row)
	c.Assert(err, qt.IsNil)

	encodedValue := mock.rows[0][1]
	c.Assert(string(encodedValue.Raw()), qt.Equals, "0x776f726c64")
}

func TestHexBlobWrapperWriteRowBINARY(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"id", "data"})

	row := []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_INT32, []byte("789")),
		sqltypes.MakeTrusted(querypb.Type_BINARY, []byte("test")),
	}

	_, err := wrapper.WriteRow(row)
	c.Assert(err, qt.IsNil)

	encodedValue := mock.rows[0][1]
	c.Assert(string(encodedValue.Raw()), qt.Equals, "0x74657374")
}

func TestHexBlobWrapperWriteRowNullBlob(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"id", "data"})

	row := []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_INT32, []byte("123")),
		sqltypes.MakeTrusted(querypb.Type_BLOB, nil),
	}

	_, err := wrapper.WriteRow(row)
	c.Assert(err, qt.IsNil)

	// NULL should pass through unchanged
	c.Assert(mock.rows[0][1].Raw(), qt.IsNil)
}

func TestHexBlobWrapperWriteRowNonBinaryTypes(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"id", "name", "age"})

	row := []sqltypes.Value{
		sqltypes.MakeTrusted(querypb.Type_INT32, []byte("123")),
		sqltypes.MakeTrusted(querypb.Type_VARCHAR, []byte("John")),
		sqltypes.MakeTrusted(querypb.Type_INT64, []byte("25")),
	}

	_, err := wrapper.WriteRow(row)
	c.Assert(err, qt.IsNil)

	// All non-binary values should pass through unchanged
	for i, origVal := range row {
		c.Assert(mock.rows[0][i].String(), qt.Equals, origVal.String())
	}
}

func TestHexBlobWrapperInitialize(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"id", "data"})

	fieldNames := []string{"col1", "col2", "col3"}
	err := wrapper.Initialize(fieldNames)
	c.Assert(err, qt.IsNil)
	c.Assert(mock.initialized, qt.IsTrue)
	c.Assert(mock.fieldNames, qt.DeepEquals, fieldNames)
}

func TestHexBlobWrapperShouldFlush(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"id", "data"})

	result := wrapper.ShouldFlush()
	c.Assert(result, qt.IsFalse)
}

func TestHexBlobWrapperFlush(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"id", "data"})

	err := wrapper.Flush("/tmp", "testdb", "testtable", 1)
	c.Assert(err, qt.IsNil)
}

func TestHexBlobWrapperClose(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"id", "data"})

	err := wrapper.Close("/tmp", "testdb", "testtable", 1)
	c.Assert(err, qt.IsNil)
}

func TestHexBlobWrapperEncodeToHexPrefix(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"data"})

	// Test that hex encoding includes 0x prefix
	result := wrapper.encodeToHex([]byte("\x00\xff\xaa\xbb"))
	c.Assert(string(result), qt.Equals, "0x00ffaabb")
}

func TestHexBlobWrapperEncodeToHexEmpty(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"data"})

	result := wrapper.encodeToHex([]byte{})
	c.Assert(string(result), qt.Equals, "X''")
}

func TestHexBlobWrapperShouldHexEncodeDetectsNull(t *testing.T) {
	c := qt.New(t)

	mock := &mockTableWriter{}
	wrapper := NewHexBlobWrapper(mock, []string{"data"})

	nullValue := sqltypes.MakeTrusted(querypb.Type_BLOB, nil)
	result := wrapper.shouldHexEncode(nullValue)
	c.Assert(result, qt.IsFalse)
}
