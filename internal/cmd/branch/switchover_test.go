package branch

import (
	"bytes"
	"context"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func switchoverTestHelper(org string, dbSvc *mock.DatabaseService, svc *mock.PostgresSwitchoversService, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	if format == printer.Human {
		p.SetHumanOutput(buf)
	}
	p.SetResourceOutput(buf)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: dbSvc, PostgresSwitchovers: svc}, nil
		},
	}
}

func TestBranch_SwitchoverCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"

	res := &ps.PostgresSwitchover{
		ID:    "switchover-1",
		State: "pending",
	}

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: ps.DatabaseEnginePostgres}, nil
		},
	}
	svc := &mock.PostgresSwitchoversService{
		CreateFn: func(ctx context.Context, req *ps.CreatePostgresSwitchoverRequest) (*ps.PostgresSwitchover, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Candidate, qt.Equals, "hzi-replica-2")

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
				Databases:           dbSvc,
				PostgresSwitchovers: svc,
			}, nil
		},
	}

	cmd := SwitchoverCmd(ch)
	cmd.SetArgs([]string{db, branch, "--candidate", "hzi-replica-2"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_SwitchoverCmd_VitessDatabase(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: req.Database, Kind: ps.DatabaseEngineMySQL}, nil
		},
	}
	svc := &mock.PostgresSwitchoversService{
		CreateFn: func(ctx context.Context, req *ps.CreatePostgresSwitchoverRequest) (*ps.PostgresSwitchover, error) {
			return nil, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: "planetscale",
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:           dbSvc,
				PostgresSwitchovers: svc,
			}, nil
		},
	}

	cmd := SwitchoverCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*only available for PostgreSQL.*mysql.*`)
	c.Assert(svc.CreateFnInvoked, qt.IsFalse)
}

func TestBranch_SwitchoverListCmd_JSON(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	res := []*ps.PostgresSwitchover{
		{ID: "switchover-2", State: "running", Method: "switchover"},
		{ID: "switchover-1", State: "succeeded", Method: "restart"},
	}
	dbSvc := postgresSwitchoverDatabaseService("app", ps.DatabaseEnginePostgres)
	svc := &mock.PostgresSwitchoversService{
		ListFn: func(_ context.Context, req *ps.ListPostgresSwitchoversRequest, _ ...ps.ListOption) ([]*ps.PostgresSwitchover, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "app")
			c.Assert(req.Branch, qt.Equals, "main")
			return res, nil
		},
	}

	cmd := SwitchoverCmd(switchoverTestHelper("planetscale", dbSvc, svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"list", "app", "main", "--page", "2", "--per-page", "50"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_SwitchoverListCmd_HumanAndEmpty(t *testing.T) {
	tests := []struct {
		name string
		args []string
		data []*ps.PostgresSwitchover
		want string
	}{
		{name: "rows", args: []string{"app", "main"}, data: []*ps.PostgresSwitchover{{ID: "switchover-1", State: "succeeded", Method: "switchover"}}, want: "switchover-1"},
		{name: "empty", args: []string{"app", "main"}, data: []*ps.PostgresSwitchover{}, want: "No switchovers exist"},
		{name: "empty page", args: []string{"app", "main", "--page", "2"}, data: []*ps.PostgresSwitchover{}, want: "No switchovers found on this page."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			var buf bytes.Buffer
			svc := &mock.PostgresSwitchoversService{
				ListFn: func(context.Context, *ps.ListPostgresSwitchoversRequest, ...ps.ListOption) ([]*ps.PostgresSwitchover, error) {
					return tt.data, nil
				},
			}
			cmd := SwitchoverListCmd(switchoverTestHelper("planetscale", postgresSwitchoverDatabaseService("app", ps.DatabaseEnginePostgres), svc, printer.Human, &buf))
			cmd.SetArgs(tt.args)
			c.Assert(cmd.Execute(), qt.IsNil)
			c.Assert(buf.String(), qt.Contains, tt.want)
		})
	}
}

func TestBranch_SwitchoverListCmd_EmptyJSON(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.PostgresSwitchoversService{
		ListFn: func(context.Context, *ps.ListPostgresSwitchoversRequest, ...ps.ListOption) ([]*ps.PostgresSwitchover, error) {
			return []*ps.PostgresSwitchover{}, nil
		},
	}
	cmd := SwitchoverListCmd(switchoverTestHelper("planetscale", postgresSwitchoverDatabaseService("app", ps.DatabaseEnginePostgres), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(strings.TrimSpace(buf.String()), qt.Equals, "[]")
}

func TestBranch_SwitchoverListCmd_NotFound(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.PostgresSwitchoversService{
		ListFn: func(context.Context, *ps.ListPostgresSwitchoversRequest, ...ps.ListOption) ([]*ps.PostgresSwitchover, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}
	cmd := SwitchoverListCmd(switchoverTestHelper("planetscale", postgresSwitchoverDatabaseService("app", ps.DatabaseEnginePostgres), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"app", "missing"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `(?s).*branch.*missing.*does not exist.*`)
}

func TestBranch_SwitchoverShowCmd_JSON(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	res := &ps.PostgresSwitchover{ID: "switchover-1", State: "failed", Method: "restart", Error: "The branch is draining"}
	svc := &mock.PostgresSwitchoversService{
		GetFn: func(_ context.Context, req *ps.GetPostgresSwitchoverRequest) (*ps.PostgresSwitchover, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "app")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.ID, qt.Equals, "switchover-1")
			return res, nil
		},
	}
	cmd := SwitchoverCmd(switchoverTestHelper("planetscale", postgresSwitchoverDatabaseService("app", ps.DatabaseEnginePostgres), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"show", "app", "main", "switchover-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestBranch_SwitchoverShowCmd_Human(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	res := &ps.PostgresSwitchover{ID: "switchover-1", State: "succeeded", Method: "switchover"}
	svc := &mock.PostgresSwitchoversService{
		GetFn: func(context.Context, *ps.GetPostgresSwitchoverRequest) (*ps.PostgresSwitchover, error) {
			return res, nil
		},
	}
	cmd := SwitchoverShowCmd(switchoverTestHelper("planetscale", postgresSwitchoverDatabaseService("app", ps.DatabaseEnginePostgres), svc, printer.Human, &buf))
	cmd.SetArgs([]string{"app", "main", "switchover-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "switchover-1")
	c.Assert(buf.String(), qt.Contains, "succeeded")
}

func TestBranch_SwitchoverShowCmd_NotFound(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.PostgresSwitchoversService{
		GetFn: func(context.Context, *ps.GetPostgresSwitchoverRequest) (*ps.PostgresSwitchover, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}
	cmd := SwitchoverShowCmd(switchoverTestHelper("planetscale", postgresSwitchoverDatabaseService("app", ps.DatabaseEnginePostgres), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"app", "main", "missing"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `(?s).*switchover.*missing.*does not exist.*`)
}

func TestBranch_SwitchoverReadCmds_VitessDatabase(t *testing.T) {
	tests := []struct {
		name string
		show bool
		args []string
	}{
		{name: "list", args: []string{"app", "main"}},
		{name: "show", show: true, args: []string{"app", "main", "switchover-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			var buf bytes.Buffer
			svc := &mock.PostgresSwitchoversService{}
			ch := switchoverTestHelper("planetscale", postgresSwitchoverDatabaseService("app", ps.DatabaseEngineMySQL), svc, printer.JSON, &buf)
			cmd := SwitchoverListCmd(ch)
			if tt.show {
				cmd = SwitchoverShowCmd(ch)
			}
			cmd.SetArgs(tt.args)
			c.Assert(cmd.Execute(), qt.ErrorMatches, `(?s).*only available for PostgreSQL.*mysql.*`)
			c.Assert(svc.ListFnInvoked, qt.IsFalse)
			c.Assert(svc.GetFnInvoked, qt.IsFalse)
		})
	}
}

func postgresSwitchoverDatabaseService(name string, kind ps.DatabaseEngine) *mock.DatabaseService {
	return &mock.DatabaseService{
		GetFn: func(context.Context, *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: name, Kind: kind}, nil
		},
	}
}
