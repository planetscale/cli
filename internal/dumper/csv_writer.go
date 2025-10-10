package dumper

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/xelabs/go-mysqlstack/sqlparser/depends/sqltypes"
)

type csvWriter struct {
	cfg        *Config
	fieldNames []string
	csvBuffer  bytes.Buffer
	writer     *csv.Writer
	chunkbytes int
}

func newCSVWriter(cfg *Config) *csvWriter {
	return &csvWriter{
		cfg: cfg,
	}
}

func (w *csvWriter) Initialize(fieldNames []string) error {
	w.fieldNames = fieldNames
	w.csvBuffer.Reset()
	w.writer = csv.NewWriter(&w.csvBuffer)

	if err := w.writer.Write(fieldNames); err != nil {
		return err
	}
	w.writer.Flush()
	return nil
}

func (w *csvWriter) WriteRow(row []sqltypes.Value) (int, error) {
	csvRow := make([]string, len(row))
	for i, v := range row {
		if v.Raw() == nil {
			csvRow[i] = ""
		} else {
			csvRow[i] = v.String()
		}
	}

	if err := w.writer.Write(csvRow); err != nil {
		return 0, err
	}
	w.writer.Flush()

	rowBytes := w.csvBuffer.Len()
	bytesAdded := rowBytes - w.chunkbytes
	w.chunkbytes = rowBytes
	return bytesAdded, nil
}

func (w *csvWriter) ShouldFlush() bool {
	return (w.chunkbytes / 1024 / 1024) >= w.cfg.ChunksizeInMB
}

func (w *csvWriter) Flush(outdir, database, table string, fileNo int) error {
	file := fmt.Sprintf("%s/%s.%s.%05d.csv", outdir, database, table, fileNo)
	err := writeFile(file, w.csvBuffer.String())
	if err != nil {
		return err
	}

	w.csvBuffer.Reset()
	w.writer = csv.NewWriter(&w.csvBuffer)
	if err := w.writer.Write(w.fieldNames); err != nil {
		return err
	}
	w.writer.Flush()
	w.chunkbytes = 0
	return nil
}

func (w *csvWriter) Close(outdir, database, table string, fileNo int) error {
	if w.csvBuffer.Len() > 0 {
		return w.Flush(outdir, database, table, fileNo)
	}
	return nil
}
