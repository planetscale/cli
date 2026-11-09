package oidctrust

import (
	"bytes"
	"context"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestOIDCTrust_ShowCmd(t *testing.T) {
	c := qt.New(t)

	trust := &ps.OIDCTrust{
		ID:            "trust-123",
		DisplayName:   "CI",
		Provider:      "github",
		Audience:      "https://example.com",
		MaxTTLSeconds: 3600,
		CreatedAt:     time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Conditions: []ps.OIDCTrustCondition{
			{Claim: "repository", Value: "acme/app"},
			{Claim: "ref", Value: "refs/heads/main"},
		},
		Accesses: []ps.OIDCTrustAccess{
			{Access: "manage_passwords", ResourceType: "Organization", ResourceName: "acme"},
			{Access: "read_branch", ResourceType: "Database", ResourceName: "mydb"},
		},
		BulkDatabaseAccesses: []string{"connect_branch"},
	}

	svc := &mock.OIDCTrustsService{
		GetFn: func(ctx context.Context, req *ps.GetOIDCTrustRequest) (*ps.OIDCTrust, error) {
			c.Assert(req.Organization, qt.Equals, "acme")
			c.Assert(req.ID, qt.Equals, "trust-123")
			return trust, nil
		},
	}

	run := func(format printer.Format) string {
		var buf bytes.Buffer
		p := printer.NewPrinter(&format)
		p.SetResourceOutput(&buf)
		ch := &cmdutil.Helper{
			Printer: p,
			Config:  &config.Config{Organization: "acme"},
			Client:  func() (*ps.Client, error) { return &ps.Client{OIDCTrusts: svc}, nil },
		}
		cmd := ShowCmd(ch)
		cmd.SetArgs([]string{"trust-123"})
		c.Assert(cmd.Execute(), qt.IsNil)
		return buf.String()
	}

	c.Assert(run(printer.JSON), qt.JSONEquals, trust)

	row := toOIDCTrust(trust)
	c.Assert(row.Conditions, qt.Equals, "repository=acme/app, ref=refs/heads/main")
	c.Assert(row.Accesses, qt.Equals, "manage_passwords, mydb:read_branch, *:connect_branch")
}
