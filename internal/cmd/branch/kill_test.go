package branch

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/printer"
)

func processlistKillServer(t *testing.T, c *qt.C, wantMethod, wantPath, wantQuery, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, wantMethod)
		c.Assert(r.URL.Path, qt.Equals, wantPath)
		c.Assert(r.URL.RawQuery, qt.Equals, wantQuery)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestKill(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	server := processlistKillServer(t, c,
		http.MethodDelete,
		"/v1/organizations/my-org/databases/my-db/branches/my-branch/connections/connection/101",
		"",
		`{"success":true,"keyspace":"main","shard":"-","tablet":"zone1-2001","id":101,"kind":"connection"}`)

	var buf bytes.Buffer
	ch := processlistTestHelper(org, server.URL, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"kill", db, branch, "101"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, `"success": true`)
}

func TestConnectionsKillUsesConnectionsEndpoint(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	server := processlistKillServer(t, c,
		http.MethodDelete,
		"/v1/organizations/my-org/databases/my-db/branches/my-branch/connections/query/zone1-1001-101",
		"keyspace=commerce&shard=-80",
		`{"success":true,"keyspace":"commerce","shard":"-80","tablet":"zone1-1001","id":101,"kind":"query"}`)

	var buf bytes.Buffer
	ch := processlistTestHelper(org, server.URL, printer.JSON, &buf)

	cmd := ConnectionsCmd(ch)
	cmd.SetArgs([]string{"kill", db, branch, "zone1-1001-101", "--query", "--keyspace", "commerce", "--shard", "-80"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
}

func TestKill_CSVOutput(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	server := processlistKillServer(t, c,
		http.MethodDelete,
		"/v1/organizations/my-org/databases/my-db/branches/my-branch/connections/connection/101",
		"",
		`{"success":true,"keyspace":"main","shard":"-","tablet":"zone1-2001","id":101,"kind":"connection"}`)

	var buf bytes.Buffer
	ch := processlistTestHelper(org, server.URL, printer.CSV, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"kill", db, branch, "101"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "true,main,-,zone1-2001,101,connection")
	c.Assert(buf.String(), qt.Not(qt.Contains), "{")
	c.Assert(buf.String(), qt.Not(qt.Contains), `"success"`)
}

func TestKill_QueryFlag(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	server := processlistKillServer(t, c,
		http.MethodDelete,
		"/v1/organizations/my-org/databases/my-db/branches/my-branch/connections/query/101",
		"",
		`{"success":true,"id":101,"kind":"query"}`)

	var buf bytes.Buffer
	ch := processlistTestHelper(org, server.URL, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"kill", db, branch, "101", "--query"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
}

func TestKill_InvalidID(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	ch := processlistTestHelper("my-org", "http://127.0.0.1:1", printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"kill", "my-db", "my-branch", "not-a-number"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
}

func TestKill_NotFound(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "missing-db", "missing-branch"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/missing-db/branches/missing-branch/connections/connection/101")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"not found"}`)
	}))
	t.Cleanup(server.Close)

	var buf bytes.Buffer
	ch := processlistTestHelper(org, server.URL, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"kill", db, branch, "101"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "connection_id not found")
	c.Assert(err.Error(), qt.Not(qt.Contains), "Not Found")
}
