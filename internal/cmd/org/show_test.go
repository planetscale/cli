package org

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"testing"
	"testing/fstest"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func TestOrganization_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	configPath, err := config.DefaultConfigPath()
	c.Assert(err, qt.IsNil)

	organization := "planetscale"

	testfs := memFS{
		configPath: &fstest.MapFile{
			Data: []byte(fmt.Sprintf("org: %s", organization)),
		},
	}

	ch := &cmdutil.Helper{
		Printer:  p,
		ConfigFS: config.NewConfigFS(testfs),
	}

	cmd := ShowCmd(ch)
	err = cmd.Execute()
	c.Assert(err, qt.IsNil)

	res := map[string]string{"org": organization}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

// memFS is copied from fstest.MapFS to support absolute paths
type memFS map[string]*fstest.MapFile

func (m memFS) Open(name string) (fs.File, error) {
	file, ok := m[name]
	if !ok {
		return nil, os.ErrNotExist
	}

	return &openMapFile{name, mapFileInfo{path.Base(name), file}, 0}, nil
}

// An openMapFile is a regular (non-directory) fs.File open for reading.
type openMapFile struct {
	path string
	mapFileInfo
	offset int64
}

func (f *openMapFile) Stat() (fs.FileInfo, error) { return &f.mapFileInfo, nil }

func (f *openMapFile) Close() error { return nil }

func (f *openMapFile) Read(b []byte) (int, error) {
	if f.offset >= int64(len(f.f.Data)) {
		return 0, io.EOF
	}
	if f.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.path, Err: fs.ErrInvalid}
	}
	n := copy(b, f.f.Data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

// A mapFileInfo implements fs.FileInfo and fs.DirEntry for a given map file.
type mapFileInfo struct {
	name string
	f    *fstest.MapFile
}

func (i *mapFileInfo) Name() string               { return i.name }
func (i *mapFileInfo) Size() int64                { return int64(len(i.f.Data)) }
func (i *mapFileInfo) Mode() fs.FileMode          { return i.f.Mode }
func (i *mapFileInfo) Type() fs.FileMode          { return i.f.Mode.Type() }
func (i *mapFileInfo) ModTime() time.Time         { return i.f.ModTime }
func (i *mapFileInfo) IsDir() bool                { return i.f.Mode&fs.ModeDir != 0 }
func (i *mapFileInfo) Sys() interface{}           { return i.f.Sys }
func (i *mapFileInfo) Info() (fs.FileInfo, error) { return i, nil }
