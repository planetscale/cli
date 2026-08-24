package role

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func listQueryParam(opts []ps.ListOption, key string) string {
	lo := &ps.ListOptions{URLValues: &url.Values{}}
	for _, opt := range opts {
		_ = opt(lo)
	}
	return lo.URLValues.Get(key)
}

func TestRole_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	expiresAt := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	disabledAt := time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)
	roles := []*ps.PostgresRole{
		{Name: "reader", ExpiresAt: &expiresAt},
		{Name: "writer", DisabledAt: &disabledAt, Expired: true},
		{Name: "old-reader", Expired: true},
	}

	listCalls := 0
	svc := &mock.PostgresRolesService{
		ListFn: func(ctx context.Context, req *ps.ListPostgresRolesRequest, opts ...ps.ListOption) ([]*ps.PostgresRole, error) {
			listCalls++
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(listQueryParam(opts, "q"), qt.Equals, "analytics")
			c.Assert(listQueryParam(opts, "status"), qt.Equals, "disabled")
			c.Assert(listQueryParam(opts, "per_page"), qt.Equals, "100")
			c.Assert(listQueryParam(opts, "page"), qt.Equals, "")
			return roles, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresRoles: svc}, nil
		},
	}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{db, branch, "--name", "analytics", "--status", "disabled"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(listCalls, qt.Equals, 1)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)

	var output []struct {
		Status    string  `json:"status"`
		ExpiresAt *string `json:"expires_at"`
	}
	c.Assert(json.Unmarshal(buf.Bytes(), &output), qt.IsNil)
	c.Assert(output, qt.HasLen, 3)
	c.Assert(output[0].Status, qt.Equals, "active")
	wantExpiresAt := "2026-08-20T12:00:00Z"
	c.Assert(output[0].ExpiresAt, qt.DeepEquals, &wantExpiresAt)
	c.Assert(output[1].Status, qt.Equals, "disabled")
	c.Assert(output[1].ExpiresAt, qt.IsNil)
	c.Assert(output[2].Status, qt.Equals, "expired")
	c.Assert(output[2].ExpiresAt, qt.IsNil)
}

func TestRole_ListHumanOutputIncludesStatusAndExpiration(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	expiresAt := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	err := p.PrintResource(toPostgresRoles([]*ps.PostgresRole{
		{Name: "reader", ExpiresAt: &expiresAt},
	}))
	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "STATUS")
	c.Assert(buf.String(), qt.Contains, "EXPIRES AT")
	c.Assert(buf.String(), qt.Contains, "active")
	c.Assert(buf.String(), qt.Contains, "2026-08-20T12:00:00Z")
}

func TestRole_ListCmdFilteredEmpty(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	svc := &mock.PostgresRolesService{
		ListFn: func(context.Context, *ps.ListPostgresRolesRequest, ...ps.ListOption) ([]*ps.PostgresRole, error) {
			return nil, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PostgresRoles: svc}, nil
		},
	}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{"planetscale", "development", "--status", "expired"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "match the specified filters")
	c.Assert(buf.String(), qt.Not(qt.Contains), "No roles exist")
}
