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

func TestPostgresSwitchovers_List(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/switchovers")
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "50")
		_, err := w.Write([]byte(`{"data":[{"id":"switchover-1","state":"succeeded","method":"switchover","actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"},"started_at":"2021-01-14T10:20:00.000Z","completed_at":"2021-01-14T10:21:00.000Z","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:21:00.000Z"}]}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	got, err := client.PostgresSwitchovers.List(context.Background(), &ListPostgresSwitchoversRequest{
		Organization: testOrg, Database: "my-db", Branch: "main",
	}, WithPage(2), WithPerPage(50))
	c.Assert(err, qt.IsNil)
	c.Assert(got, qt.HasLen, 1)
	c.Assert(got[0].ID, qt.Equals, "switchover-1")
	c.Assert(got[0].Actor.Name, qt.Equals, "Alice")
}

func TestPostgresSwitchovers_ListEmptyUsesDefaultPagination(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "100")
		_, err := w.Write([]byte(`{"data":[]}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	got, err := client.PostgresSwitchovers.List(context.Background(), &ListPostgresSwitchoversRequest{
		Organization: testOrg, Database: "my-db", Branch: "main",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(got, qt.HasLen, 0)
}

func TestPostgresSwitchovers_Get(t *testing.T) {
	c := qt.New(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/main/switchovers/switchover-1")
		_, err := w.Write([]byte(`{"id":"switchover-1","state":"failed","method":"restart","error":"The branch is draining","actor":{"id":"user-1","display_name":"Alice","avatar_url":"https://example.com/a.png"},"started_at":"2021-01-14T10:20:00.000Z","completed_at":"2021-01-14T10:21:00.000Z","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:21:00.000Z"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	got, err := client.PostgresSwitchovers.Get(context.Background(), &GetPostgresSwitchoverRequest{
		Organization: testOrg, Database: "my-db", Branch: "main", ID: "switchover-1",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(got.ID, qt.Equals, "switchover-1")
	c.Assert(got.State, qt.Equals, "failed")
	c.Assert(got.Method, qt.Equals, "restart")
	c.Assert(got.Error, qt.Equals, "The branch is draining")
}
