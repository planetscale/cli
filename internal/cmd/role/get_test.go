package role

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func TestRole_GetCmdIncludesStatusAndExpiration(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	expiresAt := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	svc := &mock.PostgresRolesService{
		GetFn: func(ctx context.Context, req *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.RoleId, qt.Equals, "role-id")
			return &ps.PostgresRole{
				ID:            "role-id",
				Name:          "app",
				Username:      "app-user",
				AccessHostURL: "pg.psdb.cloud",
				ExpiresAt:     &expiresAt,
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

	cmd := GetCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "role-id"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)

	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"id":               "role-id",
		"name":             "app",
		"username":         "app-user",
		"status":           "active",
		"expires_at":       "2026-08-20T12:00:00Z",
		"password":         "",
		"access_host_url":  "pg.psdb.cloud",
		"database_url":     "postgresql://app-user:@pg.psdb.cloud:5432/postgres?sslmode=verify-full",
		"with_replication": false,
	})
}
