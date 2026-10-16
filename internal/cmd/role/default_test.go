package role

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func TestRole_DefaultCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.PostgresRolesService{
		GetDefaultRoleFn: func(ctx context.Context, req *ps.GetDefaultPostgresRoleRequest) (*ps.PostgresRole, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.Branch, qt.Equals, "main")
			return &ps.PostgresRole{
				ID:       "role-id",
				Name:     "postgres",
				Username: "postgres",
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresRoles: svc}, nil
		},
	}

	cmd := DefaultCmd(ch)
	cmd.SetArgs([]string{"mydb", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetDefaultRoleFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"username": "postgres"`)
}

func TestRole_DefaultCmd_NotCreated(t *testing.T) {
	c := qt.New(t)

	format := printer.Human
	p := printer.NewPrinter(&format)

	roles := &mock.PostgresRolesService{
		GetDefaultRoleFn: func(ctx context.Context, req *ps.GetDefaultPostgresRoleRequest) (*ps.PostgresRole, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}
	branches := &mock.PostgresBranchesService{
		GetFn: func(ctx context.Context, req *ps.GetPostgresBranchRequest) (*ps.PostgresBranch, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.Branch, qt.Equals, "main")
			return &ps.PostgresBranch{Name: "main"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresRoles: roles, PostgresBranches: branches}, nil
		},
	}

	cmd := DefaultCmd(ch)
	cmd.SetArgs([]string{"mydb", "main"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(branches.GetFnInvoked, qt.IsTrue)
	c.Assert(err.Error(), qt.Contains, "has not been created yet")
	c.Assert(err.Error(), qt.Contains, "pscale role reset-default mydb main")
	c.Assert(err.Error(), qt.Not(qt.Contains), "does not exist")
}

func TestRole_DefaultCmd_NotCreatedJSON(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	roles := &mock.PostgresRolesService{
		GetDefaultRoleFn: func(ctx context.Context, req *ps.GetDefaultPostgresRoleRequest) (*ps.PostgresRole, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}
	branches := &mock.PostgresBranchesService{
		GetFn: func(ctx context.Context, req *ps.GetPostgresBranchRequest) (*ps.PostgresBranch, error) {
			return &ps.PostgresBranch{Name: "main"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresRoles: roles, PostgresBranches: branches}, nil
		},
	}

	cmd := DefaultCmd(ch)
	cmd.SetArgs([]string{"mydb", "main"})
	err := cmd.Execute()

	var cmdErr *cmdutil.Error
	c.Assert(errors.As(err, &cmdErr), qt.IsTrue)
	c.Assert(cmdErr.ExitCode, qt.Equals, cmdutil.ActionRequestedExitCode)

	var resp cmdutil.JSONErrorResponse
	c.Assert(json.Unmarshal(buf.Bytes(), &resp), qt.IsNil)
	c.Assert(resp.Status, qt.Equals, "action_required")
	c.Assert(resp.Code(), qt.Equals, "NOT_FOUND")
	c.Assert(resp.NextSteps, qt.Contains, "pscale role reset-default mydb main --org planetscale --format json --force")
}

func TestRole_DefaultCmd_BranchNotFound(t *testing.T) {
	c := qt.New(t)

	format := printer.Human
	p := printer.NewPrinter(&format)

	roles := &mock.PostgresRolesService{
		GetDefaultRoleFn: func(ctx context.Context, req *ps.GetDefaultPostgresRoleRequest) (*ps.PostgresRole, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}
	branches := &mock.PostgresBranchesService{
		GetFn: func(ctx context.Context, req *ps.GetPostgresBranchRequest) (*ps.PostgresBranch, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresRoles: roles, PostgresBranches: branches}, nil
		},
	}

	cmd := DefaultCmd(ch)
	cmd.SetArgs([]string{"mydb", "main"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
}
