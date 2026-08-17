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

func TestDatabase_ThrottlerShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	throttler := &ps.DatabaseThrottler{
		Keyspaces: []string{"main"},
		Configurations: []*ps.ThrottlerConfiguration{
			{KeyspaceName: "main", Ratio: 50},
		},
	}

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEngineMySQL}, nil
		},
		GetThrottlerFn: func(ctx context.Context, req *ps.GetDatabaseThrottlerRequest) (*ps.DatabaseThrottler, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return throttler, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: svc}, nil
		},
	}

	cmd := ThrottlerShowCmd(ch)
	cmd.SetArgs([]string{db})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetThrottlerFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, throttler)
}

func TestDatabase_ThrottlerUpdateCmd_Ratio(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	throttler := &ps.DatabaseThrottler{
		Keyspaces: []string{"main"},
		Configurations: []*ps.ThrottlerConfiguration{
			{KeyspaceName: "main", Ratio: 25},
		},
	}

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEngineMySQL}, nil
		},
		UpdateThrottlerFn: func(ctx context.Context, req *ps.UpdateDatabaseThrottlerRequest) (*ps.DatabaseThrottler, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Ratio, qt.IsNotNil)
			c.Assert(*req.Ratio, qt.Equals, 25)
			c.Assert(req.Configurations, qt.IsNil)
			return throttler, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: svc}, nil
		},
	}

	cmd := ThrottlerUpdateCmd(ch)
	cmd.SetArgs([]string{db, "--ratio", "25"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateThrottlerFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, throttler)
}

func TestDatabase_ThrottlerUpdateCmd_Configurations(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	throttler := &ps.DatabaseThrottler{
		Keyspaces: []string{"main", "sharded"},
		Configurations: []*ps.ThrottlerConfiguration{
			{KeyspaceName: "main", Ratio: 10},
			{KeyspaceName: "sharded", Ratio: 40},
		},
	}

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEngineMySQL}, nil
		},
		UpdateThrottlerFn: func(ctx context.Context, req *ps.UpdateDatabaseThrottlerRequest) (*ps.DatabaseThrottler, error) {
			c.Assert(req.Ratio, qt.IsNil)
			c.Assert(req.Configurations, qt.DeepEquals, []*ps.UpdateThrottlerConfiguration{
				{KeyspaceName: "main", Ratio: 10},
				{KeyspaceName: "sharded", Ratio: 40},
			})
			return throttler, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: svc}, nil
		},
	}

	cmd := ThrottlerUpdateCmd(ch)
	cmd.SetArgs([]string{db, "--configuration", "main=10", "--configuration", "sharded=40"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateThrottlerFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, throttler)
}

func TestDatabase_ThrottlerUpdateCmd_RequiresFlags(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: &mock.DatabaseService{}}, nil
		},
	}

	cmd := ThrottlerUpdateCmd(ch)
	cmd.SetArgs([]string{"db"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Equals, "must specify --ratio or --configuration")
}

func TestDatabase_ThrottlerUpdateCmd_MutuallyExclusive(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: &mock.DatabaseService{}}, nil
		},
	}

	cmd := ThrottlerUpdateCmd(ch)
	cmd.SetArgs([]string{"db", "--ratio", "10", "--configuration", "main=10"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Equals, "cannot use both --ratio and --configuration; pick one mode")
}

func TestDatabase_ThrottlerShowCmd_RejectsPostgres(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)
	org := "planetscale"
	db := "eagle"

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEnginePostgres}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: svc}, nil
		},
	}

	cmd := ThrottlerShowCmd(ch)
	cmd.SetArgs([]string{db})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "only available for Vitess (MySQL) databases")
	c.Assert(err.Error(), qt.Contains, "postgresql")
	c.Assert(svc.GetThrottlerFnInvoked, qt.IsFalse)
}

func TestDatabase_ThrottlerUpdateCmd_RejectsPostgres(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)
	org := "planetscale"
	db := "eagle"

	svc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEnginePostgres}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: svc}, nil
		},
	}

	cmd := ThrottlerUpdateCmd(ch)
	cmd.SetArgs([]string{db, "--ratio", "10"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "only available for Vitess (MySQL) databases")
	c.Assert(svc.UpdateThrottlerFnInvoked, qt.IsFalse)
}
