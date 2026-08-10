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

func TestPostgresCIDRs_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/cidrs")
		w.WriteHeader(200)
		out := `{
			"data":[{
				"id":"cidr-1",
				"schema":"public",
				"role":"reader",
				"cidrs":["192.168.1.0/24","10.0.0.1/32"],
				"description":"office network",
				"created_at":"2021-01-14T10:19:23.000Z",
				"updated_at":"2021-01-14T10:19:23.000Z",
				"deleted_at":null,
				"actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"}
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	entries, err := client.PostgresCIDRs.List(context.Background(), &ListPostgresCIDRsRequest{
		Organization: testOrg,
		Database:     "my-db",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(entries, qt.HasLen, 1)
	c.Assert(entries[0].ID, qt.Equals, "cidr-1")
	c.Assert(entries[0].Schema, qt.Equals, "public")
	c.Assert(entries[0].Role, qt.Equals, "reader")
	c.Assert(entries[0].CIDRs, qt.DeepEquals, []string{"192.168.1.0/24", "10.0.0.1/32"})
	c.Assert(entries[0].Description, qt.IsNotNil)
	c.Assert(*entries[0].Description, qt.Equals, "office network")
	c.Assert(entries[0].Actor.Name, qt.Equals, "Alice")
	c.Assert(entries[0].DeletedAt, qt.IsNil)
}

func TestPostgresCIDRs_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/cidrs/cidr-1")
		w.WriteHeader(200)
		out := `{
			"id":"cidr-1",
			"schema":"",
			"role":"",
			"cidrs":["203.0.113.0/24"],
			"description":null,
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-14T10:19:23.000Z",
			"deleted_at":null,
			"actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"}
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	entry, err := client.PostgresCIDRs.Get(context.Background(), &GetPostgresCIDRRequest{
		Organization: testOrg,
		Database:     "my-db",
		ID:           "cidr-1",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(entry.ID, qt.Equals, "cidr-1")
	c.Assert(entry.Schema, qt.Equals, "")
	c.Assert(entry.Role, qt.Equals, "")
	c.Assert(entry.Description, qt.IsNil)
	c.Assert(entry.CIDRs, qt.DeepEquals, []string{"203.0.113.0/24"})
}

func TestPostgresCIDRs_Create(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/cidrs")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["schema"], qt.Equals, "public")
		c.Assert(body["role"], qt.Equals, "writer")
		c.Assert(body["cidrs"], qt.DeepEquals, []any{"192.168.1.0/24"})
		c.Assert(body["description"], qt.Equals, "vpn")

		w.WriteHeader(201)
		out := `{
			"id":"cidr-2",
			"schema":"public",
			"role":"writer",
			"cidrs":["192.168.1.0/24"],
			"description":"vpn",
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-14T10:19:23.000Z",
			"deleted_at":null,
			"actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"}
		}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	entry, err := client.PostgresCIDRs.Create(context.Background(), &CreatePostgresCIDRRequest{
		Organization: testOrg,
		Database:     "my-db",
		Schema:       "public",
		Role:         "writer",
		CIDRs:        []string{"192.168.1.0/24"},
		Description:  "vpn",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(entry.ID, qt.Equals, "cidr-2")
	c.Assert(entry.CreatedAt.Equal(time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)), qt.IsTrue)
}

func TestPostgresCIDRs_Update(t *testing.T) {
	c := qt.New(t)

	desc := "updated"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/cidrs/cidr-1")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["cidrs"], qt.DeepEquals, []any{"10.0.0.0/8"})
		c.Assert(body["description"], qt.Equals, "updated")
		_, ok := body["schema"]
		c.Assert(ok, qt.IsFalse)

		w.WriteHeader(200)
		out := `{
			"id":"cidr-1",
			"schema":"",
			"role":"",
			"cidrs":["10.0.0.0/8"],
			"description":"updated",
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-15T10:19:23.000Z",
			"deleted_at":null,
			"actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"}
		}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	entry, err := client.PostgresCIDRs.Update(context.Background(), &UpdatePostgresCIDRRequest{
		Organization: testOrg,
		Database:     "my-db",
		ID:           "cidr-1",
		CIDRs:        []string{"10.0.0.0/8"},
		Description:  &desc,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(entry.CIDRs, qt.DeepEquals, []string{"10.0.0.0/8"})
	c.Assert(*entry.Description, qt.Equals, "updated")
}

func TestPostgresCIDRs_Delete(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/cidrs/cidr-1")
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.PostgresCIDRs.Delete(context.Background(), &DeletePostgresCIDRRequest{
		Organization: testOrg,
		Database:     "my-db",
		ID:           "cidr-1",
	})
	c.Assert(err, qt.IsNil)
}
