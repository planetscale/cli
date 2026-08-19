package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestPostgresSwitchovers_Create(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/switchovers")

		var body map[string]any
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body["candidate"], qt.Equals, "hzi-replica-2")

		w.WriteHeader(201)
		out := `{
			"id":"switchover-1",
			"state":"pending",
			"actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"},
			"started_at":null,
			"completed_at":null,
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-14T10:19:23.000Z"
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	switchover, err := client.PostgresSwitchovers.Create(context.Background(), &CreatePostgresSwitchoverRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
		Candidate:    "hzi-replica-2",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(switchover.ID, qt.Equals, "switchover-1")
	c.Assert(switchover.State, qt.Equals, "pending")
	c.Assert(switchover.Method, qt.Equals, "")
	c.Assert(switchover.StartedAt, qt.IsNil)
	c.Assert(switchover.Actor.Name, qt.Equals, "Alice")
}

func TestPostgresSwitchovers_CreateWithoutCandidate(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		_, present := body["candidate"]
		c.Assert(present, qt.IsFalse)

		w.WriteHeader(201)
		_, err := w.Write([]byte(`{
			"id":"switchover-2",
			"state":"pending",
			"actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"},
			"started_at":null,
			"completed_at":null,
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-14T10:19:23.000Z"
		}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	switchover, err := client.PostgresSwitchovers.Create(context.Background(), &CreatePostgresSwitchoverRequest{
		Organization: testOrg,
		Database:     "my-db",
		Branch:       "main",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(switchover.ID, qt.Equals, "switchover-2")
}
