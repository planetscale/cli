package oauthapplication

import (
	"bytes"
	"context"
	"net/url"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestOAuthApplicationListCmd(t *testing.T) {
	c := qt.New(t)
	var output bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&output)
	application := &ps.OAuthApplication{ID: "app-123", Name: "Example", ClientID: "client-123", CreatedAt: time.Now()}
	service := &mock.OAuthApplicationsService{
		ListFn: func(ctx context.Context, req *ps.ListOAuthApplicationsRequest, opts ...ps.ListOption) ([]*ps.OAuthApplication, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			values := url.Values{}
			for _, opt := range opts {
				c.Assert(opt(&ps.ListOptions{URLValues: &values}), qt.IsNil)
			}
			c.Assert(values.Get("page"), qt.Equals, "2")
			c.Assert(values.Get("per_page"), qt.Equals, "10")
			return []*ps.OAuthApplication{application}, nil
		},
	}
	cmd := ListCmd(oauthApplicationHelper(p, "my-org", service))
	cmd.SetArgs([]string{"--page", "2", "--per-page", "10"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.ListFnInvoked, qt.IsTrue)
	c.Assert(output.String(), qt.JSONEquals, []*OAuthApplication{{orig: application}})
}
