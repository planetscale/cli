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

func TestDatabase_AggressiveCutoverShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	status := &ps.AggressiveCutover{Enabled: true}

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEngineMySQL}, nil
		},
		GetAggressiveCutoverFn: func(ctx context.Context, req *ps.AggressiveCutoverRequest) (*ps.AggressiveCutover, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return status, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: svc}, nil
		},
	}

	cmd := AggressiveCutoverShowCmd(ch)
	cmd.SetArgs([]string{db})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetAggressiveCutoverFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, status)
}

func TestDatabase_AggressiveCutoverEnableCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	status := &ps.AggressiveCutover{Enabled: true}

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEngineMySQL}, nil
		},
		EnableAggressiveCutoverFn: func(ctx context.Context, req *ps.AggressiveCutoverRequest) (*ps.AggressiveCutover, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return status, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: svc}, nil
		},
	}

	cmd := AggressiveCutoverEnableCmd(ch)
	cmd.SetArgs([]string{db})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.EnableAggressiveCutoverFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, status)
}

func TestDatabase_AggressiveCutoverDisableCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	status := &ps.AggressiveCutover{Enabled: false}

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEngineMySQL}, nil
		},
		DisableAggressiveCutoverFn: func(ctx context.Context, req *ps.AggressiveCutoverRequest) (*ps.AggressiveCutover, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return status, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: svc}, nil
		},
	}

	cmd := AggressiveCutoverDisableCmd(ch)
	cmd.SetArgs([]string{db})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.DisableAggressiveCutoverFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, status)
}

func TestDatabase_AggressiveCutoverCmd_RejectsPostgres(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	org := "planetscale"
	db := "planetscale"

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEnginePostgres}, nil
		},
		GetAggressiveCutoverFn: func(ctx context.Context, req *ps.AggressiveCutoverRequest) (*ps.AggressiveCutover, error) {
			c.Fatal("GetAggressiveCutover should not be called for Postgres")
			return nil, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: svc}, nil
		},
	}

	cmd := AggressiveCutoverShowCmd(ch)
	cmd.SetArgs([]string{db})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "only available for Vitess")
	c.Assert(svc.GetAggressiveCutoverFnInvoked, qt.IsFalse)
}
