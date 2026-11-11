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

func newUpdateHelper(svc *mock.OIDCTrustsService) *cmdutil.Helper {
	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client:  func() (*ps.Client, error) { return &ps.Client{OIDCTrusts: svc}, nil },
	}
}

func TestOIDCTrust_UpdateCmd_OnlySendsChangedFields(t *testing.T) {
	c := qt.New(t)

	svc := &mock.OIDCTrustsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateOIDCTrustRequest) (*ps.OIDCTrust, error) {
			c.Assert(req.ID, qt.Equals, "trust-1")
			c.Assert(req.Accesses, qt.DeepEquals, &[]ps.OIDCTrustAccessInput{{Access: "create_databases"}})
			c.Assert(req.Conditions, qt.IsNil)
			c.Assert(req.BulkDatabaseAccesses, qt.IsNil)
			c.Assert(req.DisplayName, qt.IsNil)
			c.Assert(req.MaxTTLSeconds, qt.IsNil)
			return &ps.OIDCTrust{ID: "trust-1"}, nil
		},
	}

	cmd := UpdateCmd(newUpdateHelper(svc))
	cmd.SetArgs([]string{"trust-1", "--org-access", "create_databases"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestOIDCTrust_UpdateCmd_NoFlags(t *testing.T) {
	c := qt.New(t)

	svc := &mock.OIDCTrustsService{}
	cmd := UpdateCmd(newUpdateHelper(svc))
	cmd.SetArgs([]string{"trust-1"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `at least one of --name.*must be provided`)
	c.Assert(svc.UpdateFnInvoked, qt.IsFalse)
}

func TestOIDCTrust_UpdateCmd_EmptyFlagsClearLists(t *testing.T) {
	c := qt.New(t)

	svc := &mock.OIDCTrustsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateOIDCTrustRequest) (*ps.OIDCTrust, error) {
			c.Assert(req.Conditions, qt.DeepEquals, &[]ps.OIDCTrustConditionInput{})
			c.Assert(req.Accesses, qt.DeepEquals, &[]ps.OIDCTrustAccessInput{})
			c.Assert(req.BulkDatabaseAccesses, qt.DeepEquals, &[]string{})
			return &ps.OIDCTrust{ID: "trust-1"}, nil
		},
	}

	cmd := UpdateCmd(newUpdateHelper(svc))
	cmd.SetArgs([]string{"trust-1", "--condition=", "--org-access=", "--database-access=", "--all-databases-access="})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}
