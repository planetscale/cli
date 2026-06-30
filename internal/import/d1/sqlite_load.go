package d1

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	execabs "golang.org/x/sys/execabs"
)

const defaultSQLiteChunkBytes = 64 << 20 // 64 MiB of SQL per .read batch

type sqliteSourceMeta struct {
	DumpSize    int64     `json:"dump_size"`
	DumpModTime time.Time `json:"dump_mod_time"`
}

// EnsureSQLiteFromDump loads dump SQL into sqlite unless a fresh-enough database already exists.
func EnsureSQLiteFromDump(ctx context.Context, dumpPath, sqlitePath string) error {
	if canReuseSQLite(dumpPath, sqlitePath) {
		return nil
	}
	return buildSQLiteFromDump(ctx, dumpPath, sqlitePath)
}

// BuildSQLiteFromDump always rebuilds sqlite from the dump (tests and forced refresh).
func BuildSQLiteFromDump(ctx context.Context, dumpPath, sqlitePath string) error {
	return buildSQLiteFromDump(ctx, dumpPath, sqlitePath)
}

func buildSQLiteFromDump(ctx context.Context, dumpPath, sqlitePath string) error {
	dumpPath, err := ValidateInputPath(dumpPath)
	if err != nil {
		return err
	}

	sqlite3, err := FindSQLite3()
	if err != nil {
		return err
	}

	if err := os.RemoveAll(sqlitePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	_ = os.Remove(sqliteSourceMetaPath(sqlitePath))

	dir := filepath.Dir(sqlitePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	if err := loadSQLiteDumpChunked(ctx, sqlite3, dumpPath, sqlitePath, defaultSQLiteChunkBytes); err != nil {
		return err
	}
	return writeSQLiteSourceMeta(dumpPath, sqlitePath)
}

func sqliteSourceMetaPath(sqlitePath string) string {
	return sqlitePath + ".source"
}

func writeSQLiteSourceMeta(dumpPath, sqlitePath string) error {
	dumpInfo, err := os.Stat(dumpPath)
	if err != nil {
		return err
	}
	meta := sqliteSourceMeta{
		DumpSize:    dumpInfo.Size(),
		DumpModTime: dumpInfo.ModTime(),
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(sqliteSourceMetaPath(sqlitePath), data, 0o600)
}

func canReuseSQLite(dumpPath, sqlitePath string) bool {
	dumpInfo, err := os.Stat(dumpPath)
	if err != nil {
		return false
	}
	sqliteInfo, err := os.Stat(sqlitePath)
	if err != nil || sqliteInfo.Size() == 0 {
		return false
	}
	if !sqliteHasTables(sqlitePath) {
		return false
	}
	metaData, err := os.ReadFile(sqliteSourceMetaPath(sqlitePath))
	if err != nil {
		return false
	}
	var meta sqliteSourceMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return false
	}
	if meta.DumpSize != dumpInfo.Size() {
		return false
	}
	if !meta.DumpModTime.Equal(dumpInfo.ModTime()) {
		return false
	}
	return true
}

func sqliteHasTables(sqlitePath string) bool {
	sqlite3, err := FindSQLite3()
	if err != nil {
		return false
	}
	out, err := execabs.Command(sqlite3, sqlitePath, "SELECT 1 FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' LIMIT 1;").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "1"
}

func loadSQLiteDumpChunked(ctx context.Context, sqlite3, dumpPath, sqlitePath string, chunkBytes int64) error {
	dump, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	defer dump.Close()

	chunkDir, err := os.MkdirTemp("", "pscale-d1-sqlite-chunk-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(chunkDir)

	reader := bufio.NewReader(dump)
	var (
		chunkIdx   int
		chunkFile  *os.File
		chunkPath  string
		chunkSize  int64
		lineNo     int
		totalLines int
	)

	flushChunk := func() error {
		if chunkFile == nil {
			return nil
		}
		if err := chunkFile.Close(); err != nil {
			return err
		}
		chunkFile = nil

		readPath := strings.ReplaceAll(chunkPath, "'", "''")
		cmd := execabs.CommandContext(ctx, sqlite3, sqlitePath, fmt.Sprintf(".read %s", readPath))
		var stderr bytes.Buffer
		cmd.Stdout = io.Discard
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf(
				"sqlite3 chunk %d (through line %d): %w: %s",
				chunkIdx,
				lineNo,
				err,
				truncateLoadError(stderr.String(), 2048),
			)
		}
		return os.Remove(chunkPath)
	}

	startChunk := func() error {
		chunkIdx++
		chunkPath = filepath.Join(chunkDir, fmt.Sprintf("chunk-%04d.sql", chunkIdx))
		f, err := os.OpenFile(chunkPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return err
		}
		chunkFile = f
		chunkSize = 0
		return nil
	}

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			lineNo++
			totalLines++
			if chunkFile == nil {
				if err := startChunk(); err != nil {
					return err
				}
			}
			if _, werr := chunkFile.Write(line); werr != nil {
				return werr
			}
			chunkSize += int64(len(line))
			if chunkSize >= chunkBytes && lineEndsSQLStatement(line) {
				if err := flushChunk(); err != nil {
					return err
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	if err := flushChunk(); err != nil {
		return fmt.Errorf("sqlite3 load failed: %w", err)
	}
	if totalLines == 0 {
		return fmt.Errorf("sqlite3 load failed: dump is empty")
	}
	return nil
}

func truncateLoadError(msg string, max int) string {
	msg = strings.TrimSpace(msg)
	if len(msg) <= max {
		return msg
	}
	return msg[:max] + "..."
}

// lineEndsSQLStatement reports whether line completes a standalone SQL statement.
// Chunk flushes use this so multi-line CREATE TABLE blocks are never split.
func lineEndsSQLStatement(line []byte) bool {
	s := strings.TrimSpace(string(line))
	if s == "" || strings.HasPrefix(s, "--") {
		return false
	}
	return sqlEndsWithSemicolon(s)
}

func sqlEndsWithSemicolon(s string) bool {
	inSingle := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\'' {
			if inSingle && i+1 < len(s) && s[i+1] == '\'' {
				i++
				continue
			}
			inSingle = !inSingle
			continue
		}
		if c == ';' && !inSingle && strings.TrimSpace(s[i+1:]) == "" {
			return true
		}
	}
	return false
}
