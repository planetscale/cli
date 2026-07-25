package dumper

import (
	"encoding/hex"

	querypb "github.com/xelabs/go-mysqlstack/sqlparser/depends/query"
	"github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"
)

// hexBlobWrapper wraps a TableWriter and hex-encodes BLOB columns
type hexBlobWrapper struct {
	underlying TableWriter
	fieldNames []string
}

// NewHexBlobWrapper creates a wrapper that hex-encodes BLOB columns for the underlying writer
func NewHexBlobWrapper(underlying TableWriter, fieldNames []string) *hexBlobWrapper {
	return &hexBlobWrapper{
		underlying: underlying,
		fieldNames: fieldNames,
	}
}

// Initialize passes through to the underlying writer
func (w *hexBlobWrapper) Initialize(fieldNames []string) error {
	w.fieldNames = fieldNames
	return w.underlying.Initialize(fieldNames)
}

// WriteRow hex-encodes BLOB values before passing to the underlying writer
func (w *hexBlobWrapper) WriteRow(row []sqltypes.Value) (int, error) {
	hexRow := make([]sqltypes.Value, len(row))
	for i, v := range row {
		if w.shouldHexEncode(v) {
			hexRow[i] = sqltypes.NewVarBinary(string(w.encodeToHex(v.Raw())))
		} else {
			hexRow[i] = v
		}
	}

	return w.underlying.WriteRow(hexRow)
}

// shouldHexEncode determines if a value should be hex-encoded (BLOB/BINARY types)
func (w *hexBlobWrapper) shouldHexEncode(v sqltypes.Value) bool {
	if v.Raw() == nil {
		return false
	}

	vType := v.Type()
	switch vType {
	case querypb.Type_BLOB,
		querypb.Type_BINARY,
		querypb.Type_VARBINARY:
		return true
	}

	return false
}

// encodeToHex converts raw bytes to hex with 0x prefix (MySQL format)
func (w *hexBlobWrapper) encodeToHex(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("X''")
	}

	encoded := hex.EncodeToString(raw)
	// Prefix with 0x for MySQL compatibility
	return append([]byte("0x"), []byte(encoded)...)
}

func (w *hexBlobWrapper) ShouldFlush() bool {
	return w.underlying.ShouldFlush()
}

func (w *hexBlobWrapper) Flush(outdir, database, table string, fileNo int) error {
	return w.underlying.Flush(outdir, database, table, fileNo)
}

func (w *hexBlobWrapper) Close(outdir, database, table string, fileNo int) error {
	return w.underlying.Close(outdir, database, table, fileNo)
}
