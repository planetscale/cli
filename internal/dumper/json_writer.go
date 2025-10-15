package dumper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"
)

type jsonWriter struct {
	cfg        *Config
	fieldNames []string
	jsonLines  []string
	chunkbytes int
}

func newJSONWriter(cfg *Config) *jsonWriter {
	return &jsonWriter{
		cfg:       cfg,
		jsonLines: make([]string, 0, 256),
	}
}

func (w *jsonWriter) Initialize(fieldNames []string) error {
	w.fieldNames = fieldNames
	return nil
}

func (w *jsonWriter) WriteRow(row []sqltypes.Value) (int, error) {
	rowMap := make(map[string]interface{})
	for i, v := range row {
		if v.Raw() == nil {
			rowMap[w.fieldNames[i]] = nil
		} else {
			rowMap[w.fieldNames[i]] = v.String()
		}
	}

	jsonBytes, err := json.Marshal(rowMap)
	if err != nil {
		return 0, err
	}

	jsonLine := string(jsonBytes) + "\n"
	w.jsonLines = append(w.jsonLines, jsonLine)

	lineBytes := len(jsonLine)
	w.chunkbytes += lineBytes
	return lineBytes, nil
}

func (w *jsonWriter) ShouldFlush() bool {
	return (w.chunkbytes / 1024 / 1024) >= w.cfg.ChunksizeInMB
}

func (w *jsonWriter) Flush(outdir, database, table string, fileNo int) error {
	file := fmt.Sprintf("%s/%s.%s.%05d.json", outdir, database, table, fileNo)
	err := writeFile(file, strings.Join(w.jsonLines, ""))
	if err != nil {
		return err
	}

	w.jsonLines = w.jsonLines[:0]
	w.chunkbytes = 0
	return nil
}

func (w *jsonWriter) Close(outdir, database, table string, fileNo int) error {
	if len(w.jsonLines) > 0 {
		return w.Flush(outdir, database, table, fileNo)
	}
	return nil
}
