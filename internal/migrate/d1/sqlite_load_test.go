package d1

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildSQLiteFromDump(t *testing.T) {
	dir := t.TempDir()
	dumpPath := filepath.Join(dir, "dump.sql")
	sqlitePath := filepath.Join(dir, "load.sqlite")

	var b strings.Builder
	b.WriteString("PRAGMA defer_foreign_keys=TRUE;\n")
	b.WriteString("CREATE TABLE attachments (\n")
	b.WriteString("  id INTEGER PRIMARY KEY,\n")
	b.WriteString("  payload BLOB\n")
	b.WriteString(");\n")
	hex := strings.Repeat("41", 48000) // ~96 KiB blob, similar to wrangler export lines
	fmt.Fprintf(&b, "INSERT INTO attachments (id, payload) VALUES(1,X'%s');\n", hex)
	b.WriteString("INSERT INTO attachments (id, payload) VALUES(2,NULL);\n")

	if err := os.WriteFile(dumpPath, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := BuildSQLiteFromDump(context.Background(), dumpPath, sqlitePath); err != nil {
		t.Fatalf("BuildSQLiteFromDump: %v", err)
	}

	counts, err := CountSQLiteRows(context.Background(), sqlitePath, []string{"attachments"})
	if err != nil {
		t.Fatal(err)
	}
	if counts["attachments"] != 2 {
		t.Fatalf("expected 2 rows, got %d", counts["attachments"])
	}
}

func TestEnsureSQLiteFromDumpReusesExisting(t *testing.T) {
	dir := t.TempDir()
	dumpPath := filepath.Join(dir, "dump.sql")
	sqlitePath := filepath.Join(dir, "load.sqlite")

	content := "PRAGMA defer_foreign_keys=TRUE;\nCREATE TABLE t (id INTEGER PRIMARY KEY);\nINSERT INTO t VALUES(1);\n"
	if err := os.WriteFile(dumpPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := BuildSQLiteFromDump(context.Background(), dumpPath, sqlitePath); err != nil {
		t.Fatal(err)
	}
	info1, err := os.Stat(sqlitePath)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(dumpPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	// Touch dump to be newer than sqlite — should rebuild.
	dumpInfo, _ := os.Stat(dumpPath)
	if err := os.Chtimes(dumpPath, dumpInfo.ModTime().Add(time.Second), dumpInfo.ModTime().Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	if err := EnsureSQLiteFromDump(context.Background(), dumpPath, sqlitePath); err != nil {
		t.Fatal(err)
	}
	info2, err := os.Stat(sqlitePath)
	if err != nil {
		t.Fatal(err)
	}
	if !info2.ModTime().After(info1.ModTime()) {
		t.Fatal("expected rebuild when dump is newer than sqlite")
	}

	// Dump must not be newer than sqlite for reuse.
	now := time.Now()
	if err := os.Chtimes(dumpPath, now.Add(-2*time.Minute), now.Add(-2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(sqlitePath, now.Add(-time.Minute), now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	if err := EnsureSQLiteFromDump(context.Background(), dumpPath, sqlitePath); err != nil {
		t.Fatal(err)
	}
	info3, err := os.Stat(sqlitePath)
	if err != nil {
		t.Fatal(err)
	}
	if info3.ModTime().After(info2.ModTime()) {
		t.Fatal("expected sqlite reuse without rebuild when dump is not newer")
	}
}

func TestLoadSQLiteDumpChunked(t *testing.T) {
	dir := t.TempDir()
	dumpPath := filepath.Join(dir, "multi.sql")
	sqlitePath := filepath.Join(dir, "multi.sqlite")

	var b strings.Builder
	b.WriteString("PRAGMA defer_foreign_keys=TRUE;\n")
	b.WriteString("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT);\n")
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&b, "INSERT INTO t (id, v) VALUES(%d,'row');\n", i)
	}
	if err := os.WriteFile(dumpPath, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	sqlite3, err := FindSQLite3()
	if err != nil {
		t.Fatal(err)
	}
	// Force many small chunks to exercise batching.
	if err := loadSQLiteDumpChunked(context.Background(), sqlite3, dumpPath, sqlitePath, 256); err != nil {
		t.Fatalf("loadSQLiteDumpChunked: %v", err)
	}

	counts, err := CountSQLiteRows(context.Background(), sqlitePath, []string{"t"})
	if err != nil {
		t.Fatal(err)
	}
	if counts["t"] != 200 {
		t.Fatalf("expected 200 rows, got %d", counts["t"])
	}
}
