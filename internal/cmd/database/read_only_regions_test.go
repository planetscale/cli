package database

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func TestDatabase_ReadOnlyRegionsCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "analytics"
	res := []*ps.ReadOnlyRegion{
		{
			ID:          "ror123",
			DisplayName: "Europe West",
			Ready:       true,
			Region:      ps.Region{Slug: "eu-west", Name: "EU West"},
		},
	}

	svc := &mock.ReadOnlyRegionsService{
		ListFn: func(ctx context.Context, req *ps.ListReadOnlyRegionsRequest, opts ...ps.ListOption) ([]*ps.ReadOnlyRegion, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return res, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				ReadOnlyRegions: svc,
			}, nil
		},
	}

	cmd := ReadOnlyRegionsCmd(ch)
	cmd.SetArgs([]string{db})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDump_ReadOnlyRegionFlagConflicts(t *testing.T) {
	c := qt.New(t)

	format := printer.Human
	p := printer.NewPrinter(&format)
	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: "planetscale",
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{}, nil
		},
	}

	cmd := DumpCmd(ch)
	cmd.SetArgs([]string{"db", "main", "--read-only-region", "eu-west", "--rdonly"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "cannot be combined")

	cmd = DumpCmd(ch)
	cmd.SetArgs([]string{"db", "main", "--read-only-region", "eu-west", "--replica"})
	err = cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "cannot be combined")
}
