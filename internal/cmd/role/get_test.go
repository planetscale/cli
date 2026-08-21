package role

import (
	"bytes"
	"context"
	"strings"
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

func TestRole_GetCmdConnectionTargets(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		request ps.GetPostgresRoleRequest
	}{
		{
			name: "replica",
			args: []string{"mydb", "main", "role-id", "--replica"},
			request: ps.GetPostgresRoleRequest{
				Replica: true,
			},
		},
		{
			name: "read-only replica",
			args: []string{"mydb", "main", "role-id", "--read-only-replica", "us-west"},
			request: ps.GetPostgresRoleRequest{
				ReadOnlyReplica: "us-west",
			},
		},
		{
			name: "bouncer",
			args: []string{"mydb", "main", "role-id", "--bouncer", "pool"},
			request: ps.GetPostgresRoleRequest{
				Bouncer: "pool",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := qt.New(t)

			var buf bytes.Buffer
			format := printer.JSON
			p := printer.NewPrinter(&format)
			p.SetResourceOutput(&buf)

			svc := &mock.PostgresRolesService{
				GetFn: func(ctx context.Context, req *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error) {
					test.request.Organization = "planetscale"
					test.request.Database = "mydb"
					test.request.Branch = "main"
					test.request.RoleId = "role-id"
					c.Assert(req, qt.DeepEquals, &test.request)

					return &ps.PostgresRole{
						ID:            "role-id",
						Name:          "app",
						Username:      "app.branch|replica",
						AccessHostURL: "us-west.pg.psdb.cloud",
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
			cmd.SetArgs(test.args)
			c.Assert(cmd.Execute(), qt.IsNil)

			c.Assert(buf.String(), qt.JSONEquals, map[string]any{
				"id":               "role-id",
				"name":             "app",
				"username":         "app.branch|replica",
				"status":           "active",
				"expires_at":       nil,
				"password":         "",
				"access_host_url":  "us-west.pg.psdb.cloud",
				"database_url":     "postgresql://app.branch%7Creplica:@us-west.pg.psdb.cloud:5432/postgres?sslmode=verify-full",
				"with_replication": false,
			})
		})
	}
}

func TestRole_GetCmdRejectsMultipleConnectionTargets(t *testing.T) {
	c := qt.New(t)

	svc := &mock.PostgresRolesService{}
	ch := &cmdutil.Helper{
		Config: &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresRoles: svc}, nil
		},
	}

	cmd := GetCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "role-id", "--replica", "--bouncer", "pool"})

	c.Assert(cmd.Execute(), qt.IsNotNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
}

func TestRole_GetCmdConnectionTargetNotFound(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		targetType string
		target     string
	}{
		{
			name:       "read-only replica",
			args:       []string{"mydb", "main", "role-id", "--read-only-replica", "missing-region"},
			targetType: "read-only replica in region",
			target:     "missing-region",
		},
		{
			name:       "bouncer",
			args:       []string{"mydb", "main", "role-id", "--bouncer", "missing-bouncer"},
			targetType: "PgBouncer",
			target:     "missing-bouncer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := qt.New(t)

			format := printer.Human
			p := printer.NewPrinter(&format)
			svc := &mock.PostgresRolesService{
				GetFn: func(ctx context.Context, req *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error) {
					return nil, &ps.Error{Code: ps.ErrNotFound}
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
			cmd.SetArgs(test.args)
			err := cmd.Execute()

			c.Assert(err, qt.IsNotNil)
			c.Assert(strings.Contains(err.Error(), "role"), qt.IsTrue)
			c.Assert(strings.Contains(err.Error(), "role-id"), qt.IsTrue)
			c.Assert(strings.Contains(err.Error(), test.targetType), qt.IsTrue)
			c.Assert(strings.Contains(err.Error(), test.target), qt.IsTrue)
		})
	}
}
