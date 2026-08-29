package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestUsers_GetCurrentUser(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/user")

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"id": "user-123",
			"display_name": "Ada Lovelace",
			"name": "Ada",
			"email": "ada@example.com",
			"avatar_url": "https://example.com/avatar.png",
			"two_factor_auth_configured": true,
			"created_at": "2021-01-14T10:19:23.000Z",
			"updated_at": "2021-01-15T10:19:23.000Z",
			"default_organization": {
				"id": "org-123",
				"name": "acme",
				"created_at": "2021-01-14T10:19:23.000Z",
				"updated_at": "2021-01-15T10:19:23.000Z",
				"deleted_at": null
			}
		}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	user, err := client.Users.GetCurrentUser(context.Background())

	c.Assert(err, qt.IsNil)
	c.Assert(user, qt.DeepEquals, &User{
		ID:                      "user-123",
		DisplayName:             "Ada Lovelace",
		Name:                    "Ada",
		Email:                   "ada@example.com",
		AvatarURL:               "https://example.com/avatar.png",
		TwoFactorAuthConfigured: true,
		CreatedAt:               time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt:               time.Date(2021, time.January, 15, 10, 19, 23, 0, time.UTC),
		DefaultOrganization: &Organization{
			Name:      "acme",
			CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
			UpdatedAt: time.Date(2021, time.January, 15, 10, 19, 23, 0, time.UTC),
		},
	})
}
