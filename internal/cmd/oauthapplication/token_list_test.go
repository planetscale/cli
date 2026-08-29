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

func TestOAuthApplicationTokenListCmd(t *testing.T) {
	c := qt.New(t)
	var output bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&output)
	token := &ps.OAuthToken{ID: "token-123", DisplayName: "Example", CreatedAt: time.Now()}
	service := &mock.OAuthApplicationsService{
		ListTokensFn: func(ctx context.Context, req *ps.ListOAuthTokensRequest, opts ...ps.ListOption) ([]*ps.OAuthToken, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			c.Assert(req.ApplicationID, qt.Equals, "app-123")
			return []*ps.OAuthToken{token}, nil
		},
	}
	cmd := TokenListCmd(oauthApplicationHelper(p, "my-org", service))
	cmd.SetArgs([]string{"app-123"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.ListTokensFnInvoked, qt.IsTrue)
	c.Assert(output.String(), qt.JSONEquals, []*OAuthToken{{orig: token}})
}
