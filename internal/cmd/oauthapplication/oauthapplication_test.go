package oauthapplication

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func oauthApplicationHelper(p *printer.Printer, org string, service *mock.OAuthApplicationsService) *cmdutil.Helper {
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{OAuthApplications: service}, nil
		},
	}
}

func TestOAuthApplicationListCmd(t *testing.T) {
	c := qt.New(t)
	var output bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&output)
	application := &ps.OAuthApplication{ID: "app-123", Name: "Example", ClientID: "client-123", CreatedAt: time.Now()}
	// The service receives the options; apply them to verify pagination without
	// coupling this command test to the HTTP client.
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

func TestOAuthApplicationTokenCommands(t *testing.T) {
	c := qt.New(t)
	token := &ps.OAuthToken{ID: "token-123", DisplayName: "Example", CreatedAt: time.Now()}
	t.Run("list", func(t *testing.T) {
		var output bytes.Buffer
		format := printer.JSON
		p := printer.NewPrinter(&format)
		p.SetResourceOutput(&output)
		service := &mock.OAuthApplicationsService{
			ListTokensFn: func(ctx context.Context, req *ps.ListOAuthTokensRequest, opts ...ps.ListOption) ([]*ps.OAuthToken, error) {
				c.Assert(req.ApplicationID, qt.Equals, "app-123")
				return []*ps.OAuthToken{token}, nil
			},
		}
		cmd := TokenListCmd(oauthApplicationHelper(p, "my-org", service))
		cmd.SetArgs([]string{"app-123"})
		c.Assert(cmd.Execute(), qt.IsNil)
		c.Assert(output.String(), qt.JSONEquals, []*OAuthToken{{orig: token}})
	})
	t.Run("show", func(t *testing.T) {
		var output bytes.Buffer
		format := printer.JSON
		p := printer.NewPrinter(&format)
		p.SetResourceOutput(&output)
		service := &mock.OAuthApplicationsService{
			GetTokenFn: func(ctx context.Context, req *ps.GetOAuthTokenRequest) (*ps.OAuthToken, error) {
				c.Assert(req.ApplicationID, qt.Equals, "app-123")
				c.Assert(req.TokenID, qt.Equals, "token-123")
				return token, nil
			},
		}
		cmd := TokenShowCmd(oauthApplicationHelper(p, "my-org", service))
		cmd.SetArgs([]string{"app-123", "token-123"})
		c.Assert(cmd.Execute(), qt.IsNil)
		c.Assert(output.String(), qt.JSONEquals, &OAuthToken{orig: token})
	})
}

func TestOAuthApplicationTokenDeleteCmd(t *testing.T) {
	c := qt.New(t)
	var output bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&output)
	service := &mock.OAuthApplicationsService{
		DeleteTokenFn: func(ctx context.Context, req *ps.DeleteOAuthTokenRequest) error {
			c.Assert(req.TokenID, qt.Equals, "token-123")
			return nil
		},
	}
	cmd := TokenDeleteCmd(oauthApplicationHelper(p, "my-org", service))
	cmd.SetArgs([]string{"app-123", "token-123", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(output.String(), qt.JSONEquals, map[string]string{"result": "oauth token deleted"})

	output.Reset()
	service.DeleteTokenFnInvoked = false
	cmd = TokenDeleteCmd(oauthApplicationHelper(p, "my-org", service))
	cmd.SetArgs([]string{"app-123", "token-123"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, `cannot delete oauth token with the output format "json" .*`)
	c.Assert(service.DeleteTokenFnInvoked, qt.IsFalse)
}

func TestOAuthApplicationTokenCreateCmd(t *testing.T) {
	c := qt.New(t)
	var output bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&output)
	secret := "client-secret"
	tokenSecret := "oauth-token"
	refreshSecret := "oauth-refresh"
	token := &ps.OAuthToken{ID: "token-123", Token: &tokenSecret, PlainTextRefreshToken: &refreshSecret}
	service := &mock.OAuthApplicationsService{
		CreateTokenFn: func(ctx context.Context, req *ps.CreateOAuthTokenRequest) (*ps.OAuthToken, error) {
			c.Assert(req.ClientID, qt.Equals, "client-123")
			c.Assert(req.ClientSecret, qt.Equals, secret)
			c.Assert(req.GrantType, qt.Equals, "authorization_code")
			return token, nil
		},
	}
	cmd := TokenCreateCmd(oauthApplicationHelper(p, "my-org", service))
	cmd.SetArgs([]string{"app-123", "--client-id", "client-123", "--client-secret", secret, "--code", "code", "--redirect-uri", "https://example.com/callback"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(output.String(), qt.JSONEquals, &OAuthTokenWithSecret{orig: token})
	c.Assert(output.String(), qt.Not(qt.Contains), secret)

	for _, test := range []struct {
		name string
		args []string
	}{
		{"invalid grant type", []string{"--grant-type", "invalid"}},
		{"authorization code missing values", []string{}},
		{"authorization code rejects refresh token", []string{"--code", "code", "--redirect-uri", "uri", "--refresh-token", "refresh"}},
		{"refresh token missing value", []string{"--grant-type", "refresh_token"}},
		{"refresh token rejects code", []string{"--grant-type", "refresh_token", "--refresh-token", "refresh", "--code", "code"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			local := TokenCreateCmd(oauthApplicationHelper(p, "my-org", &mock.OAuthApplicationsService{
				CreateTokenFn: func(context.Context, *ps.CreateOAuthTokenRequest) (*ps.OAuthToken, error) {
					t.Fatal("API should not be called")
					return nil, nil
				},
			}))
			args := append([]string{"app-123", "--client-id", "client-123", "--client-secret", "secret"}, test.args...)
			local.SetArgs(args)
			err := local.Execute()
			c.Assert(err, qt.IsNotNil)
		})
	}

	cmd = TokenCreateCmd(oauthApplicationHelper(p, "my-org", service))
	cmd.SetIn(strings.NewReader(secret))
	cmd.SetArgs([]string{"app-123", "--client-id", "client-123", "--client-secret", "@-", "--code", "code", "--redirect-uri", "uri"})
	c.Assert(cmd.Execute(), qt.IsNil)
}
