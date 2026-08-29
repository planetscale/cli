package oauthapplication

import (
	"bytes"
	"context"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

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

	stdinService := &mock.OAuthApplicationsService{
		CreateTokenFn: func(ctx context.Context, req *ps.CreateOAuthTokenRequest) (*ps.OAuthToken, error) {
			c.Assert(req.ClientSecret, qt.Equals, "secret")
			return token, nil
		},
	}
	cmd = TokenCreateCmd(oauthApplicationHelper(p, "my-org", stdinService))
	cmd.SetIn(strings.NewReader("secret\n"))
	cmd.SetArgs([]string{"app-123", "--client-id", "client-123", "--client-secret", "@-", "--code", "code", "--redirect-uri", "uri"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(stdinService.CreateTokenFnInvoked, qt.IsTrue)

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
}
