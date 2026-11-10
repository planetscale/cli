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
		ID:            "trust-1",
		DisplayName:   "CI",
		Provider:      "github",
		Audience:      "planetscale",
		MaxTTLSeconds: 3600,
		CreatedAt:     time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Conditions: []ps.OIDCTrustCondition{
			{Claim: "repository", Value: "acme/app"},
			{Claim: "ref", Value: "refs/heads/main"},
		},
		Accesses: []ps.OIDCTrustAccess{
			{Access: "create_databases", ResourceType: "Organization", ResourceName: "planetscale"},
			{Access: "connect_branch", ResourceType: "Database", ResourceName: "mydb"},
		},
		BulkDatabaseAccesses: []string{"read_branch"},
	}

	svc := &mock.OIDCTrustsService{
		GetFn: func(ctx context.Context, req *ps.GetOIDCTrustRequest) (*ps.OIDCTrust, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.ID, qt.Equals, "trust-1")
			return trust, nil
		},
	}

	run := func(format printer.Format) string {
		var buf bytes.Buffer
		p := printer.NewPrinter(&format)
		p.SetResourceOutput(&buf)
		ch := &cmdutil.Helper{
			Printer: p,
			Config:  &config.Config{Organization: "planetscale"},
			Client:  func() (*ps.Client, error) { return &ps.Client{OIDCTrusts: svc}, nil },
		}
		cmd := ShowCmd(ch)
		cmd.SetArgs([]string{"trust-1"})
		c.Assert(cmd.Execute(), qt.IsNil)
		return buf.String()
	}

	c.Assert(run(printer.JSON), qt.JSONEquals, &OIDCTrust{orig: trust})

	table := run(printer.Human)
	c.Assert(table, qt.Contains, "repository=acme/app,ref=refs/heads/main")
	c.Assert(table, qt.Contains, "create_databases,mydb:connect_branch,*:read_branch")
}
