package oidctrust

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestOIDCTrust_DeleteCmd_RequiresForceInJSON(t *testing.T) {
	c := qt.New(t)

	svc := &mock.OIDCTrustsService{}
	cmd := DeleteCmd(newUpdateHelper(svc))
	cmd.SetArgs([]string{"trust-1"})
	c.Assert(cmd.Execute(), qt.Not(qt.IsNil))
	c.Assert(svc.DeleteFnInvoked, qt.IsFalse)
}

func TestOIDCTrust_DeleteCmd_Force(t *testing.T) {
	c := qt.New(t)

	svc := &mock.OIDCTrustsService{
		DeleteFn: func(ctx context.Context, req *ps.DeleteOIDCTrustRequest) error {
			c.Assert(req.ID, qt.Equals, "trust-1")
			return nil
		},
	}
	cmd := DeleteCmd(newUpdateHelper(svc))
	cmd.SetArgs([]string{"trust-1", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
}
