package oauthapplication

import (
	"bytes"
	"context"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestOAuthApplicationTokenShowCmd(t *testing.T) {
	c := qt.New(t)
	var output bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&output)
	token := &ps.OAuthToken{ID: "token-123", DisplayName: "Example", CreatedAt: time.Now()}
	service := &mock.OAuthApplicationsService{
		GetTokenFn: func(ctx context.Context, req *ps.GetOAuthTokenRequest) (*ps.OAuthToken, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			c.Assert(req.ApplicationID, qt.Equals, "app-123")
			c.Assert(req.TokenID, qt.Equals, "token-123")
			return token, nil
		},
	}
	cmd := TokenShowCmd(oauthApplicationHelper(p, "my-org", service))
	cmd.SetArgs([]string{"app-123", "token-123"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.GetTokenFnInvoked, qt.IsTrue)
	c.Assert(output.String(), qt.JSONEquals, &OAuthToken{orig: token})
}
