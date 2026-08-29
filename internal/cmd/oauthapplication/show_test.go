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

func TestOAuthApplicationShowCmd(t *testing.T) {
	c := qt.New(t)
	var output bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&output)
	application := &ps.OAuthApplication{ID: "app-123", Name: "Example"}
	service := &mock.OAuthApplicationsService{
		GetFn: func(ctx context.Context, req *ps.GetOAuthApplicationRequest) (*ps.OAuthApplication, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			c.Assert(req.ID, qt.Equals, "app-123")
			return application, nil
		},
	}
	cmd := ShowCmd(oauthApplicationHelper(p, "my-org", service))
	cmd.SetArgs([]string{"app-123"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(output.String(), qt.JSONEquals, &OAuthApplication{orig: application})
}
