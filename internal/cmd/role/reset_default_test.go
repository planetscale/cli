package role

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	qt "github.com/frankban/quicktest"
)

func TestResetDefaultCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "testdb"
	branch := "main"

	expectedRole := &ps.PostgresRole{
		ID:            "role-123",
		Name:          "postgres",
		Username:      "postgres",
		Password:      "new-password-123",
		AccessHostURL: "pg.psdb.cloud",
	}

	svc := &mock.PostgresRolesService{
		ResetDefaultRoleFn: func(ctx context.Context, req *ps.ResetDefaultRoleRequest) (*ps.PostgresRole, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			return expectedRole, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				PostgresRoles: svc,
			}, nil
		},
	}

	cmd := ResetDefaultCmd(ch)
	cmd.SetArgs([]string{db, branch, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ResetDefaultRoleFnInvoked, qt.IsTrue)

	expectedOutput := map[string]string{
		"id":              "role-123",
		"name":            "postgres",
		"username":        "postgres",
		"password":        "new-password-123",
		"access_host_url": "pg.psdb.cloud",
		"database_url":    "postgresql://postgres:new-password-123@pg.psdb.cloud:5432/postgres?sslmode=verify-full",
	}
	c.Assert(buf.String(), qt.JSONEquals, expectedOutput)
}
