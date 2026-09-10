package role

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestRoleStatus(t *testing.T) {
	disabledAt := time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		role *ps.PostgresRole
		want string
	}{
		{name: "active", role: &ps.PostgresRole{}, want: "active"},
		{name: "expired", role: &ps.PostgresRole{Expired: true}, want: "expired"},
		{
			name: "disabled takes precedence over expired",
			role: &ps.PostgresRole{DisabledAt: &disabledAt, Expired: true},
			want: "disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			c.Assert(postgresRoleStatus(tt.role), qt.Equals, tt.want)
		})
	}
}

func TestPrintRoleListJSON(t *testing.T) {
	expiresAt := time.Date(2026, time.August, 20, 17, 30, 0, 0, time.UTC)
	disabledAt := time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		role          *ps.PostgresRole
		wantReady     *bool
		wantStatus    string
		wantExpiresAt any
	}{
		{
			name: "Postgres omits ready",
			role: &ps.PostgresRole{
				Type:       "BranchRole",
				ID:         "postgres-role",
				Ready:      true,
				DisabledAt: &disabledAt,
				Expired:    true,
				ExpiresAt:  &expiresAt,
			},
			wantStatus:    "disabled",
			wantExpiresAt: "2026-08-20T17:30:00Z",
		},
		{
			name: "Neki pending and expired without expiration",
			role: &ps.PostgresRole{
				Type:    "NekiRole",
				ID:      "pending-neki-role",
				Ready:   false,
				Expired: true,
			},
			wantReady:     boolPointer(false),
			wantStatus:    "expired",
			wantExpiresAt: nil,
		},
		{
			name: "Neki ready and active",
			role: &ps.PostgresRole{
				Type:      "NekiRole",
				ID:        "ready-neki-role",
				Ready:     true,
				ExpiresAt: &expiresAt,
			},
			wantReady:     boolPointer(true),
			wantStatus:    "active",
			wantExpiresAt: "2026-08-20T17:30:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			var buf bytes.Buffer
			format := printer.JSON
			p := printer.NewPrinter(&format)
			p.SetResourceOutput(&buf)

			err := printRoleList(p, []*ps.PostgresRole{tt.role})

			c.Assert(err, qt.IsNil)
			var output []map[string]any
			c.Assert(json.Unmarshal(buf.Bytes(), &output), qt.IsNil)
			c.Assert(output, qt.HasLen, 1)
			c.Assert(output[0]["status"], qt.Equals, tt.wantStatus)
			c.Assert(output[0]["expires_at"], qt.Equals, tt.wantExpiresAt)
			ready, hasReady := output[0]["ready"]
			if tt.wantReady == nil {
				c.Assert(hasReady, qt.IsFalse)
			} else {
				c.Assert(hasReady, qt.IsTrue)
				c.Assert(ready, qt.Equals, *tt.wantReady)
			}
		})
	}
}

func TestPrintRoleListHuman(t *testing.T) {
	expiresAt := time.Date(2026, time.August, 20, 17, 30, 0, 0, time.UTC)

	tests := []struct {
		name             string
		role             *ps.PostgresRole
		wantReadyColumn  bool
		wantOutputValues []string
	}{
		{
			name: "Postgres",
			role: &ps.PostgresRole{
				Type:      "BranchRole",
				ID:        "postgres-role",
				ExpiresAt: &expiresAt,
			},
			wantOutputValues: []string{"STATUS", "EXPIRES AT", "active", "2026-08-20T17:30:00Z"},
		},
		{
			name: "Neki",
			role: &ps.PostgresRole{
				Type:    "NekiRole",
				ID:      "neki-role",
				Ready:   false,
				Expired: true,
			},
			wantReadyColumn:  true,
			wantOutputValues: []string{"STATUS", "EXPIRES AT", "expired", "No"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			var buf bytes.Buffer
			format := printer.Human
			p := printer.NewPrinter(&format)
			p.SetResourceOutput(&buf)

			err := printRoleList(p, []*ps.PostgresRole{tt.role})

			c.Assert(err, qt.IsNil)
			output := buf.String()
			for _, value := range tt.wantOutputValues {
				c.Assert(output, qt.Contains, value)
			}
			c.Assert(strings.Contains(output, "READY"), qt.Equals, tt.wantReadyColumn)
		})
	}
}

func TestRoleGetOutput(t *testing.T) {
	expiresAt := time.Date(2026, time.August, 20, 17, 30, 0, 0, time.UTC)

	tests := []struct {
		name       string
		format     printer.Format
		role       *ps.PostgresRole
		wantReady  *bool
		wantValues []string
	}{
		{
			name:   "Postgres JSON",
			format: printer.JSON,
			role: &ps.PostgresRole{
				Type:      "BranchRole",
				ID:        "postgres-role",
				Ready:     true,
				ExpiresAt: &expiresAt,
			},
			wantValues: []string{"active", "2026-08-20T17:30:00Z"},
		},
		{
			name:   "Neki JSON",
			format: printer.JSON,
			role: &ps.PostgresRole{
				Type:    "NekiRole",
				ID:      "neki-role",
				Ready:   false,
				Expired: true,
			},
			wantReady:  boolPointer(false),
			wantValues: []string{"expired"},
		},
		{
			name:   "Postgres human",
			format: printer.Human,
			role: &ps.PostgresRole{
				Type:      "BranchRole",
				ID:        "postgres-role",
				ExpiresAt: &expiresAt,
			},
			wantValues: []string{"STATUS", "EXPIRES AT", "active", "2026-08-20T17:30:00Z"},
		},
		{
			name:   "Neki human",
			format: printer.Human,
			role: &ps.PostgresRole{
				Type:  "NekiRole",
				ID:    "neki-role",
				Ready: true,
			},
			wantReady:  boolPointer(true),
			wantValues: []string{"READY", "STATUS", "EXPIRES AT", "Yes", "active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			var resourceOutput bytes.Buffer
			format := tt.format
			p := printer.NewPrinter(&format)
			p.SetHumanOutput(&bytes.Buffer{})
			p.SetResourceOutput(&resourceOutput)

			svc := &mock.PostgresRolesService{
				GetFn: func(context.Context, *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error) {
					return tt.role, nil
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
			cmd.SetArgs([]string{"database", "main", tt.role.ID})
			err := cmd.Execute()

			c.Assert(err, qt.IsNil)
			output := resourceOutput.String()
			for _, value := range tt.wantValues {
				c.Assert(output, qt.Contains, value)
			}
			if tt.format == printer.JSON {
				var decoded map[string]any
				c.Assert(json.Unmarshal(resourceOutput.Bytes(), &decoded), qt.IsNil)
				ready, hasReady := decoded["ready"]
				if tt.wantReady == nil {
					c.Assert(hasReady, qt.IsFalse)
				} else {
					c.Assert(hasReady, qt.IsTrue)
					c.Assert(ready, qt.Equals, *tt.wantReady)
				}
			}
		})
	}
}

func TestPrintPostgresRoleCredentialsKeepsLifecycleFieldsOut(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	printPostgresRoleCredentials(p, &PostgresRole{
		PublicID:  "role-id",
		Status:    "active",
		ExpiresAt: stringPointer("2026-08-20T17:30:00Z"),
	})

	output := buf.String()
	c.Assert(output, qt.Contains, "ID")
	c.Assert(output, qt.Not(qt.Contains), "STATUS")
	c.Assert(output, qt.Not(qt.Contains), "EXPIRES")
	c.Assert(output, qt.Not(qt.Contains), "READY")
}

func boolPointer(value bool) *bool {
	return &value
}

func stringPointer(value string) *string {
	return &value
}
