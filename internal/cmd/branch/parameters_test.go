package branch

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestBranch_ParametersCmd(t *testing.T) {
	c := qt.New(t)
	requiresUpdate := "cluster"

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "postgres-db"
	branch := "main"

	pgSvc := &mock.PostgresBranchesService{
		ListParametersFn: func(ctx context.Context, req *ps.ListPostgresParametersRequest) ([]*ps.PostgresParameter, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Extension, qt.IsNil)
			c.Assert(req.Internal, qt.IsNil)

			return []*ps.PostgresParameter{
				{Namespace: "pgconf", Name: "max_connections", Value: "200", DefaultValue: "100", ParameterType: "integer", Restart: true, RequiresUpdate: &requiresUpdate},
				{Namespace: "pgbouncer", Name: "default_pool_size", Value: "20", DefaultValue: "20", ParameterType: "integer"},
				{Namespace: "pgconf", Name: "archive_timeout", Internal: true},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresBranches: pgSvc}, nil
		},
	}

	cmd := ParametersCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(pgSvc.ListParametersFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "max_connections")
	c.Assert(buf.String(), qt.Contains, "default_pool_size")
	var parameters []map[string]interface{}
	c.Assert(json.Unmarshal(buf.Bytes(), &parameters), qt.IsNil)
	c.Assert(parameters[0]["requires_update"], qt.Equals, "cluster")
	c.Assert(parameters[1]["requires_update"], qt.IsNil)
	c.Assert(parameters[0]["internal"], qt.Equals, false)
	c.Assert(parameters[2]["internal"], qt.Equals, true)
	c.Assert(buf.String(), qt.Contains, `"requires_update": null`)
}

func TestBranch_ParametersCmd_ListSubcommand(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	pgSvc := &mock.PostgresBranchesService{
		ListParametersFn: func(ctx context.Context, req *ps.ListPostgresParametersRequest) ([]*ps.PostgresParameter, error) {
			return []*ps.PostgresParameter{
				{Namespace: "pgconf", Name: "max_connections", Value: "200"},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: "planetscale",
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresBranches: pgSvc}, nil
		},
	}

	cmd := ParametersCmd(ch)
	cmd.SetArgs([]string{"list", "postgres-db", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(pgSvc.ListParametersFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "max_connections")
}

func TestBranch_ParametersCmd_NamespaceFilter(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	pgSvc := &mock.PostgresBranchesService{
		ListParametersFn: func(ctx context.Context, req *ps.ListPostgresParametersRequest) ([]*ps.PostgresParameter, error) {
			return []*ps.PostgresParameter{
				{Namespace: "pgconf", Name: "max_connections"},
				{Namespace: "pgbouncer", Name: "default_pool_size"},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: "planetscale",
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresBranches: pgSvc}, nil
		},
	}

	cmd := ParametersCmd(ch)
	cmd.SetArgs([]string{"postgres-db", "main", "--namespace", "pgbouncer"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "default_pool_size")
	c.Assert(buf.String(), qt.Not(qt.Contains), "max_connections")
}

func TestBranch_ParametersTable(t *testing.T) {
	c := qt.New(t)
	var parameters []*ps.PostgresParameter
	c.Assert(json.Unmarshal([]byte(`[
		{"namespace":"pgconf","name":"locked","internal":true,"requires_update":"cluster"},
		{"namespace":"pgconf","name":"editable","requires_update":null}
	]`), &parameters), qt.IsNil)

	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&out)
	c.Assert(p.PrintResource(toPostgresParameters(parameters)), qt.IsNil)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	c.Assert(len(lines), qt.Equals, 4)
	c.Assert(lines[0], qt.Contains, "REQUIRES UPDATE")
	c.Assert(lines[0], qt.Contains, "INTERNAL")
	c.Assert(strings.Fields(lines[2]), qt.DeepEquals, []string{"pgconf", "locked", "No", "Yes", "Yes"})
	c.Assert(strings.Fields(lines[3]), qt.DeepEquals, []string{"pgconf", "editable", "No", "No", "No"})
}
