package oidctrust

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

func TestParseConditionsAndAccesses(t *testing.T) {
	c := qt.New(t)

	conds, err := parseConditions([]string{"repository=acme/app", "sub=a=b", "empty="})
	c.Assert(err, qt.IsNil)
	c.Assert(conds, qt.DeepEquals, []ps.OIDCTrustConditionInput{
		{Claim: "repository", Value: "acme/app"},
		{Claim: "sub", Value: "a=b"},
		{Claim: "empty", Value: ""},
	})

	for _, bad := range []string{"noequals", "=value"} {
		_, err = parseConditions([]string{bad})
		c.Assert(err, qt.ErrorMatches, `invalid --condition ".*": expected claim=value`)
	}

	accesses, err := parseAccesses(nil, nil)
	c.Assert(err, qt.IsNil)
	c.Assert(accesses, qt.DeepEquals, []ps.OIDCTrustAccessInput{})

	accesses, err = parseAccesses([]string{"create_databases"}, []string{"mydb:connect_branch", "db:a:b"})
	c.Assert(err, qt.IsNil)
	c.Assert(accesses, qt.DeepEquals, []ps.OIDCTrustAccessInput{
		{Access: "create_databases"},
		{Access: "connect_branch", Database: "mydb"},
		{Access: "a:b", Database: "db"},
	})

	for _, bad := range []string{"nocolon", ":access", "db:"} {
		_, err = parseAccesses(nil, []string{bad})
		c.Assert(err, qt.ErrorMatches, `invalid --database-access ".*": expected database:access`)
	}
}

func TestOIDCTrust_CreateCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.OIDCTrustsService{
		CreateFn: func(ctx context.Context, req *ps.CreateOIDCTrustRequest) (*ps.OIDCTrust, error) {
			c.Assert(req, qt.DeepEquals, &ps.CreateOIDCTrustRequest{
				Organization:  "planetscale",
				DisplayName:   "CI",
				Provider:      "github",
				MaxTTLSeconds: 3600,
				Conditions: []ps.OIDCTrustConditionInput{
					{Claim: "repository", Value: "acme/app"},
					{Claim: "ref", Value: "refs/heads/main"},
				},
				Accesses: []ps.OIDCTrustAccessInput{
					{Access: "create_databases"},
					{Access: "connect_branch", Database: "mydb"},
				},
				BulkDatabaseAccesses: []string{"read_branch"},
			})
			return &ps.OIDCTrust{ID: "trust-1"}, nil
		},
	}

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client:  func() (*ps.Client, error) { return &ps.Client{OIDCTrusts: svc}, nil },
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{
		"--name", "CI", "--provider", "github", "--max-ttl", "3600",
		"--condition", "repository=acme/app", "--condition", "ref=refs/heads/main",
		"--org-access", "create_databases",
		"--database-access", "mydb:connect_branch",
		"--all-databases-access", "read_branch",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}
