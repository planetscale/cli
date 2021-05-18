package org

import (
	"bytes"
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/testutil"

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

	testfs := testutil.MemFS{
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
