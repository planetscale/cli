package role

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
