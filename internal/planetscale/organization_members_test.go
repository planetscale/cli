package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestOrganizations_ListMembers(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/members")
		c.Assert(r.URL.Query().Get("q"), qt.Equals, "ada")
		w.WriteHeader(200)
		out := `{"data":[{"id":"mem-1","role":"admin","user":{"id":"user-1","name":"Ada","email":"ada@example.com","display_name":"Ada","avatar_url":"","two_factor_auth_configured":true,"created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"},"created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	members, err := client.Organizations.ListMembers(context.Background(), &ListOrganizationMembersRequest{
		Organization: "my-org",
		Query:        "ada",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(members, qt.HasLen, 1)
	c.Assert(members[0].Role, qt.Equals, "admin")
	c.Assert(members[0].User.ID, qt.Equals, "user-1")
	c.Assert(members[0].User.Email, qt.Equals, "ada@example.com")
}

func TestOrganizations_UpdateMember(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/members/user-1")
		var body struct {
			Role string `json:"role"`
		}
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body.Role, qt.Equals, "member")
		w.WriteHeader(200)
		out := `{"id":"mem-1","role":"member","user":{"id":"user-1","name":"Ada","email":"ada@example.com","display_name":"Ada","avatar_url":"","two_factor_auth_configured":false,"created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"},"created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	member, err := client.Organizations.UpdateMember(context.Background(), &UpdateOrganizationMemberRequest{
		Organization: "my-org",
		UserID:       "user-1",
		Role:         "member",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(member.Role, qt.Equals, "member")
	c.Assert(member.UpdatedAt, qt.Equals, time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC))
}

func TestOrganizations_RemoveMember(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/members/user-1")
		c.Assert(r.URL.Query().Get("delete_passwords"), qt.Equals, "true")
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.Organizations.RemoveMember(context.Background(), &RemoveOrganizationMemberRequest{
		Organization:    "my-org",
		UserID:          "user-1",
		DeletePasswords: true,
	})
	c.Assert(err, qt.IsNil)
}
