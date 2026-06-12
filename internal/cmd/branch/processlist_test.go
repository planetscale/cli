package branch

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func processlistTestHelper(org, baseURL string, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)
	p.SetHumanOutput(buf)

	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{AccessToken: "token", Organization: org, BaseURL: baseURL},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: databaseServiceForEngine(org, ps.DatabaseEngineMySQL)}, nil
		},
	}
}

func processlistListServer(t *testing.T, c *qt.C, wantQuery string, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/connections")
		c.Assert(r.URL.RawQuery, qt.Equals, wantQuery)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

func processlistListBody(keyspace, shard, tablet, connections string) string {
	return `{"type":"list","database_kind":"mysql","next_page":null,"prev_page":null,"captured_at":"2026-06-04T12:30:00Z","instances":[],"topology":{"keyspace":"` + keyspace + `","shard":"` + shard + `","tablet":"` + tablet + `"},"data":[` + connections + `]}`
}

func TestProcesslist(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	server := processlistListServer(t, c, "keyspace=commerce&shard=-80",
		processlistListBody("commerce", "-80", "zone1-1001", `{"pid":101,"instance":"zone1-1001","usename":"vt_app","state":"Query","duration_ms":42000,"connection_id":"101","query_id":"101","query_text":"SELECT 1"}`))

	var buf bytes.Buffer
	ch := processlistTestHelper(org, server.URL, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"show", db, branch, "--keyspace", "commerce", "--shard", "-80"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, `"tablet": "zone1-1001"`)
	c.Assert(buf.String(), qt.Contains, `"username": "vt_app"`)
	c.Assert(buf.String(), qt.Contains, `"connection_id": "101"`)
}

func TestProcesslist_CSVOutput(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	server := processlistListServer(t, c, "",
		processlistListBody("commerce", "-80", "zone1-1001", `{"pid":101,"instance":"zone1-1001","usename":"vt_app","client_addr":"10.0.0.1","datname":"main","state":"Query/running","duration_ms":42000,"connection_id":"101","query_id":"101","query_text":"SELECT 1"}`))

	var buf bytes.Buffer
	ch := processlistTestHelper(org, server.URL, printer.CSV, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"show", db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "keyspace,shard,tablet")
	c.Assert(buf.String(), qt.Contains, "commerce,-80,zone1-1001,101,zone1-1001,,Query/running,42000")
	c.Assert(buf.String(), qt.Contains, "vt_app,,main,10.0.0.1,SELECT 1")
	c.Assert(buf.String(), qt.Not(qt.Contains), "{")
	c.Assert(buf.String(), qt.Not(qt.Contains), `"processes"`)
}

func TestProcesslist_NoTargetFlags(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	server := processlistListServer(t, c, "",
		processlistListBody("main", "-", "zone1-2001", ""))

	var buf bytes.Buffer
	ch := processlistTestHelper(org, server.URL, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"show", db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
}

func TestProcesslist_HumanOutputDoesNotAbbreviateNumericFields(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	server := processlistListServer(t, c, "",
		processlistListBody("main", "-", "zone1-2001", `{"pid":121500,"instance":"zone1-2001","usename":"vt_app","state":"Sleep","duration_ms":2100000,"connection_id":"121500","query_id":"121500"}`))

	var buf bytes.Buffer
	ch := processlistTestHelper(org, server.URL, printer.Human, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"show", db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "pid:             121500")
	c.Assert(buf.String(), qt.Contains, "duration:        35m0s")
	c.Assert(buf.String(), qt.Contains, "connection_id:   121500")
	c.Assert(buf.String(), qt.Contains, "121500")
	c.Assert(buf.String(), qt.Not(qt.Contains), "121.5K")
	c.Assert(buf.String(), qt.Not(qt.Contains), "2.1K")
}

func TestProcesslist_NotFound(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "missing-db", "missing-branch"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/missing-db/branches/missing-branch/connections")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"not found"}`)
	}))
	t.Cleanup(server.Close)

	var buf bytes.Buffer
	ch := processlistTestHelper(org, server.URL, printer.JSON, &buf)

	cmd := ProcesslistCmd(ch)
	cmd.SetArgs([]string{"show", db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "branch")
	c.Assert(err.Error(), qt.Contains, branch)
	c.Assert(err.Error(), qt.Contains, db)
	c.Assert(err.Error(), qt.Contains, org)
	c.Assert(err.Error(), qt.Not(qt.Contains), "Not Found")
}
