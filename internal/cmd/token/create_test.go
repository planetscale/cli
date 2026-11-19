package token

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestServiceToken_CreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	id := "123456"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	orig := &ps.ServiceToken{ID: id, CreatedAt: createdAt}

	svc := &mock.ServiceTokenService{
		CreateFn: func(ctx context.Context, req *ps.CreateServiceTokenRequest) (*ps.ServiceToken, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Name, qt.IsNil)
			c.Assert(req.CIDRs, qt.IsNil)
			return orig, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				ServiceTokens: svc,
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)

	res := &ServiceTokenWithSecret{orig: orig}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestServiceToken_CreateCmdWithName(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	id := "123456"
	name := "my-token"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	orig := &ps.ServiceToken{ID: id, Name: &name, CreatedAt: createdAt}

	svc := &mock.ServiceTokenService{
		CreateFn: func(ctx context.Context, req *ps.CreateServiceTokenRequest) (*ps.ServiceToken, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Name, qt.IsNotNil)
			c.Assert(*req.Name, qt.Equals, name)
			return orig, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				ServiceTokens: svc,
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{"--name", name})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)

	res := &ServiceTokenWithSecret{orig: orig}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestServiceToken_CreateCmdWithCIDRs(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	id := "123456"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	cidrs := []string{"10.0.0.0/8", "192.168.1.1/32"}

	orig := &ps.ServiceToken{ID: id, CreatedAt: createdAt, CIDRs: cidrs}

	svc := &mock.ServiceTokenService{
		CreateFn: func(ctx context.Context, req *ps.CreateServiceTokenRequest) (*ps.ServiceToken, error) {
			c.Assert(req.CIDRs, qt.IsNotNil)
			c.Assert(*req.CIDRs, qt.DeepEquals, cidrs)
			return orig, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{ServiceTokens: svc}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{"--cidrs", "10.0.0.0/8,192.168.1.1/32"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &ServiceTokenWithSecret{orig: orig})
}

func TestServiceToken_CreateCmdWithTTL(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	id := "123456"
	name := "my-token"
	ttl := 3600
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	expiresAt := time.Date(2025, 1, 15, 11, 30, 0, 0, time.UTC)

	orig := &ps.ServiceToken{ID: id, Name: &name, CreatedAt: createdAt, ExpiresAt: &expiresAt}

	svc := &mock.ServiceTokenService{
		CreateFn: func(ctx context.Context, req *ps.CreateServiceTokenRequest) (*ps.ServiceToken, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Name, qt.IsNotNil)
			c.Assert(*req.Name, qt.Equals, name)
			c.Assert(req.TTL, qt.IsNotNil)
			c.Assert(*req.TTL, qt.Equals, ttl)
			return orig, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				ServiceTokens: svc,
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{"--name", name, "--ttl", "3600"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)

	res := &ServiceTokenWithSecret{orig: orig}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestServiceToken_DuplicateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	sourceID := "123456"
	name := "copy-token"
	secret := "d2980bbd91a4ab878601ef0573a7af7b1b15e705"
	newID := "654321"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	var requests int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/planetscale/service-tokens/123456/duplicate")
		c.Assert(r.Header.Get("Authorization"), qt.Equals, "Bearer test-token")
		body, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(string(body), qt.Equals, "{\"name\":\"copy-token\"}\n")

		_, err = io.WriteString(w, `{"id":"654321","type":"ServiceToken","token":"d2980bbd91a4ab878601ef0573a7af7b1b15e705","name":"copy-token","created_at":"2025-01-15T10:30:00.000Z","cidrs":["203.0.113.10/32"]}`)
		c.Assert(err, qt.IsNil)
	}))
	t.Cleanup(ts.Close)

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
			AccessToken:  "test-token",
		},
		Client: func() (*ps.Client, error) {
			return ps.NewClient(ps.WithBaseURL(ts.URL), ps.WithAccessToken("test-token"))
		},
	}

	cmd := TokenCmd(ch)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"duplicate", sourceID, "--name", name, "--org", org})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(requests, qt.Equals, 1)

	tokenName := name
	orig := &ps.ServiceToken{
		ID:        newID,
		Type:      "ServiceToken",
		Token:     secret,
		Name:      &tokenName,
		CreatedAt: createdAt,
		CIDRs:     []string{"203.0.113.10/32"},
	}
	c.Assert(buf.String(), qt.JSONEquals, &ServiceTokenWithSecret{orig: orig})
}

func TestServiceToken_UpdateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	id := "123456"
	name := "renamed"
	secret := "d2980bbd91a4ab878601ef0573a7af7b1b15e705"

	var requests int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/planetscale/service-tokens/123456")
		body, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(string(body), qt.Equals, "{\"name\":\"renamed\",\"ttl\":3600}\n")
		c.Assert(string(body), qt.Not(qt.Contains), secret)
		c.Assert(string(body), qt.Not(qt.Contains), "token")

		_, err = io.WriteString(w, `{"id":"123456","type":"ServiceToken","token":"d2980bbd91a4ab878601ef0573a7af7b1b15e705","name":"renamed","created_at":"2025-01-15T10:30:00.000Z","expires_at":"2025-01-15T11:30:00.000Z"}`)
		c.Assert(err, qt.IsNil)
	}))
	t.Cleanup(ts.Close)

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
			AccessToken:  "test-token",
		},
		Client: func() (*ps.Client, error) {
			return ps.NewClient(ps.WithBaseURL(ts.URL), ps.WithAccessToken("test-token"))
		},
	}

	cmd := TokenCmd(ch)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"update", id, "--name", name, "--ttl", "3600", "--org", org})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(requests, qt.Equals, 1)
	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"id":                         id,
		"name":                       name,
		"created_at":                 "2025-01-15T10:30:00Z",
		"last_used_at":               nil,
		"last_used_ip":               nil,
		"last_used_asn_organization": nil,
		"expires_at":                 "2025-01-15T11:30:00Z",
	})
	c.Assert(buf.String(), qt.Not(qt.Contains), secret)
}
