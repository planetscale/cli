package branch

import (
	"bytes"
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func testResizeHelper(p *printer.Printer, org string, dbSvc *mock.DatabaseService, pgSvc *mock.PostgresBranchesService) *cmdutil.Helper {
	return &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:        dbSvc,
				PostgresBranches: pgSvc,
			}, nil
		},
	}
}

func TestBranch_ResizeCmd_WithParameters(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "postgres-db"
	branch := "main"

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: db, Kind: "postgresql"}, nil
		},
	}

	pgSvc := &mock.PostgresBranchesService{
		ListParametersFn: func(ctx context.Context, req *ps.ListPostgresParametersRequest) ([]*ps.PostgresParameter, error) {
			c.Assert(req.Internal, qt.IsNotNil)
			c.Assert(*req.Internal, qt.IsFalse)

			return []*ps.PostgresParameter{
				{Namespace: "pgconf", Name: "max_connections", ParameterType: "integer", Restart: true},
			}, nil
		},
		ResizeFn: func(ctx context.Context, req *ps.ResizePostgresBranchRequest) (*ps.PostgresBranchClusterResizeRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.ClusterSize, qt.Equals, "")
			c.Assert(req.Replicas, qt.IsNil)
			c.Assert(req.Parameters, qt.DeepEquals, map[string]map[string]string{
				"pgconf": {"max_connections": "200"},
			})

			return &ps.PostgresBranchClusterResizeRequest{
				ID:    "change-1",
				State: "queued",
				Parameters: map[string]map[string]any{
					"pgconf": {"max_connections": "200"},
				},
			}, nil
		},
	}

	ch := testResizeHelper(p, org, dbSvc, pgSvc)

	cmd := ResizeCmd(ch)
	cmd.SetArgs([]string{db, branch, "--parameters", "pgconf.max_connections=200"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(pgSvc.ListParametersFnInvoked, qt.IsTrue)
	c.Assert(pgSvc.ResizeFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"id": "change-1"`)
}

func TestBranch_ResizeCmd_UnknownParameter(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: "postgres-db", Kind: "postgresql"}, nil
		},
	}

	pgSvc := &mock.PostgresBranchesService{
		ListParametersFn: func(ctx context.Context, req *ps.ListPostgresParametersRequest) ([]*ps.PostgresParameter, error) {
			return []*ps.PostgresParameter{
				{Namespace: "pgconf", Name: "max_connections"},
			}, nil
		},
		ResizeFn: func(ctx context.Context, req *ps.ResizePostgresBranchRequest) (*ps.PostgresBranchClusterResizeRequest, error) {
			return nil, nil
		},
	}

	ch := testResizeHelper(p, "planetscale", dbSvc, pgSvc)

	cmd := ResizeCmd(ch)
	cmd.SetArgs([]string{"postgres-db", "main", "--parameters", "pgconf.does_not_exist=42"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist or cannot be changed")
	c.Assert(err.Error(), qt.Contains, "--org planetscale")
	c.Assert(pgSvc.ResizeFnInvoked, qt.IsFalse)
}

func TestBranch_ResizeCmd_NoOpPrintsJSON(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: "postgres-db", Kind: "postgresql"}, nil
		},
	}

	pgSvc := &mock.PostgresBranchesService{
		// A nil change request models the API's 204 No Content response.
		ResizeFn: func(ctx context.Context, req *ps.ResizePostgresBranchRequest) (*ps.PostgresBranchClusterResizeRequest, error) {
			return nil, nil
		},
	}

	ch := testResizeHelper(p, "planetscale", dbSvc, pgSvc)

	cmd := ResizeCmd(ch)
	cmd.SetArgs([]string{"postgres-db", "main", "--cluster-size", "PS_10_GCP_X86"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, `"result": "no_change"`)
	c.Assert(buf.String(), qt.Contains, `"branch": "main"`)
}

func TestBranch_ResizeCmd_NothingToChange(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	ch := testResizeHelper(p, "planetscale", &mock.DatabaseService{}, &mock.PostgresBranchesService{})

	cmd := ResizeCmd(ch)
	cmd.SetArgs([]string{"postgres-db", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "nothing to change")
}

func TestBranch_ResizeCmd_MySQLDatabase(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: "mysql-db", Kind: "mysql"}, nil
		},
	}

	ch := testResizeHelper(p, "planetscale", dbSvc, &mock.PostgresBranchesService{})

	cmd := ResizeCmd(ch)
	cmd.SetArgs([]string{"mysql-db", "main", "--cluster-size", "PS_10"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "pscale keyspace resize")
}

func TestParseParameterSets(t *testing.T) {
	c := qt.New(t)

	parameters, err := parseParameterSets([]string{
		"pgconf.max_connections=200",
		"pgconf.shared_preload_libraries=pg_stat_statements,auto_explain",
		"pgbouncer.default_pool_size=50",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(parameters, qt.DeepEquals, map[string]map[string]string{
		"pgconf": {
			"max_connections":          "200",
			"shared_preload_libraries": "pg_stat_statements,auto_explain",
		},
		"pgbouncer": {
			"default_pool_size": "50",
		},
	})

	_, err = parseParameterSets([]string{"max_connections=200"})
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "namespace")

	_, err = parseParameterSets([]string{"pgconf.max_connections"})
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "namespace.name=value")

	parameters, err = parseParameterSets(nil)
	c.Assert(err, qt.IsNil)
	c.Assert(parameters, qt.IsNil)
}
