package planetscale

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

const oauthApplicationJSON = `{
	"id":"app-123","name":"Example","redirect_uri":"https://example.com/callback","domain":"example.com",
	"created_at":"2021-01-14T10:19:23Z","updated_at":"2021-01-15T10:19:23Z","scopes":"read_database",
	"avatar":"https://example.com/avatar.png","client_id":"client-123","tokens":2,"dcr":true,
	"single_org_authorization":false,"requires_org_scope":true,"scopes_by_resource":{"database":{}},
	"all_scopes_by_resource":{"database":[]},"mcp_tool_groups":null
}`

const oauthTokenJSON = `{
	"id":"token-123","name":null,"display_name":"Example token","token":"secret-token",
	"plain_text_refresh_token":"refresh-token","avatar_url":"https://example.com/avatar.png",
	"created_at":"2021-01-14T10:19:23Z","updated_at":"2021-01-15T10:19:23Z",
	"expires_at":null,"last_used_at":"2021-01-15T10:19:23Z","actor_id":null,
	"actor_display_name":"Example User","actor_type":"user","service_token_accesses":null,
	"oauth_accesses_by_resource":{"database":{}}
}`

func newOAuthApplicationsClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	client, err := NewClient(WithBaseURL(ts.URL))
	qt.New(t).Assert(err, qt.IsNil)
	return client
}

func TestOAuthApplications_List(t *testing.T) {
	c := qt.New(t)
	client := newOAuthApplicationsClient(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/oauth-applications")
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "10")
		_, _ = io.WriteString(w, `{"data":[`+oauthApplicationJSON+`]}`)
	})
	applications, err := client.OAuthApplications.List(context.Background(), &ListOAuthApplicationsRequest{Organization: "my-org"}, WithPage(2), WithPerPage(10))
	c.Assert(err, qt.IsNil)
	c.Assert(len(applications), qt.Equals, 1)
	c.Assert(applications[0].ID, qt.Equals, "app-123")
	c.Assert(applications[0].Tokens, qt.Equals, 2)
	c.Assert(applications[0].CreatedAt, qt.Equals, time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC))
}

func TestOAuthApplications_Get(t *testing.T) {
	c := qt.New(t)
	client := newOAuthApplicationsClient(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/oauth-applications/app-123")
		_, _ = io.WriteString(w, oauthApplicationJSON)
	})
	application, err := client.OAuthApplications.Get(context.Background(), &GetOAuthApplicationRequest{Organization: "my-org", ID: "app-123"})
	c.Assert(err, qt.IsNil)
	c.Assert(application.ClientID, qt.Equals, "client-123")
	c.Assert(application.ScopesByResource, qt.Not(qt.IsNil))
}

func TestOAuthApplications_ListTokens(t *testing.T) {
	c := qt.New(t)
	client := newOAuthApplicationsClient(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/oauth-applications/app-123/tokens")
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "3")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "5")
		_, _ = io.WriteString(w, `{"data":[`+oauthTokenJSON+`]}`)
	})
	tokens, err := client.OAuthApplications.ListTokens(context.Background(), &ListOAuthTokensRequest{Organization: "my-org", ApplicationID: "app-123"}, WithPage(3), WithPerPage(5))
	c.Assert(err, qt.IsNil)
	c.Assert(len(tokens), qt.Equals, 1)
	c.Assert(tokens[0].DisplayName, qt.Equals, "Example token")
	c.Assert(tokens[0].ActorDisplayName, qt.IsNotNil)
}

func TestOAuthApplications_GetToken(t *testing.T) {
	c := qt.New(t)
	client := newOAuthApplicationsClient(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/oauth-applications/app-123/tokens/token-123")
		_, _ = io.WriteString(w, oauthTokenJSON)
	})
	token, err := client.OAuthApplications.GetToken(context.Background(), &GetOAuthTokenRequest{Organization: "my-org", ApplicationID: "app-123", TokenID: "token-123"})
	c.Assert(err, qt.IsNil)
	c.Assert(token.ID, qt.Equals, "token-123")
	c.Assert(*token.Token, qt.Equals, "secret-token")
}

func TestOAuthApplications_DeleteToken(t *testing.T) {
	c := qt.New(t)
	client := newOAuthApplicationsClient(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/oauth-applications/app-123/tokens/token-123")
		w.WriteHeader(http.StatusNoContent)
	})
	err := client.OAuthApplications.DeleteToken(context.Background(), &DeleteOAuthTokenRequest{Organization: "my-org", ApplicationID: "app-123", TokenID: "token-123"})
	c.Assert(err, qt.IsNil)
}

func TestOAuthApplications_CreateToken(t *testing.T) {
	c := qt.New(t)
	client := newOAuthApplicationsClient(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/oauth-applications/app-123/token")
		var body map[string]string
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body, qt.DeepEquals, map[string]string{
			"client_id": "client-123", "client_secret": "client-secret",
			"grant_type": "authorization_code", "code": "code-123", "redirect_uri": "https://example.com/callback",
		})
		_, _ = io.WriteString(w, oauthTokenJSON)
	})
	token, err := client.OAuthApplications.CreateToken(context.Background(), &CreateOAuthTokenRequest{
		Organization: "my-org", ID: "app-123", ClientID: "client-123", ClientSecret: "client-secret",
		GrantType: "authorization_code", Code: "code-123", RedirectURI: "https://example.com/callback",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(token.PlainTextRefreshToken, qt.IsNotNil)
}
