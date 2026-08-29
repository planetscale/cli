package oauthapplication

import (
	"bytes"
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestOAuthApplicationTokenDeleteCmd(t *testing.T) {
	c := qt.New(t)
	var output bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&output)
	service := &mock.OAuthApplicationsService{
		DeleteTokenFn: func(ctx context.Context, req *ps.DeleteOAuthTokenRequest) error {
			c.Assert(req.Organization, qt.Equals, "my-org")
			c.Assert(req.ApplicationID, qt.Equals, "app-123")
			c.Assert(req.TokenID, qt.Equals, "token-123")
			return nil
		},
	}
	cmd := TokenDeleteCmd(oauthApplicationHelper(p, "my-org", service))
	cmd.SetArgs([]string{"app-123", "token-123", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.DeleteTokenFnInvoked, qt.IsTrue)
	c.Assert(output.String(), qt.JSONEquals, map[string]string{"result": "oauth token deleted"})

	output.Reset()
	service.DeleteTokenFnInvoked = false
	cmd = TokenDeleteCmd(oauthApplicationHelper(p, "my-org", service))
	cmd.SetArgs([]string{"app-123", "token-123"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, `cannot delete oauth token with the output format "json" .*`)
	c.Assert(service.DeleteTokenFnInvoked, qt.IsFalse)
}
