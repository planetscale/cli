package branch

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func TestConnectionsCmdConstruction(t *testing.T) {
	c := qt.New(t)

	cmd := ConnectionsCmd(connectionsTestHelper("acme", ps.DatabaseEngineMySQL, nil, "http://example.invalid", printer.JSON, &bytes.Buffer{}))

	c.Assert(cmd.Use, qt.Equals, "connections <command>")
	c.Assert(cmd.Aliases, qt.HasLen, 0)
	names := commandNames(cmd)
	for _, name := range []string{"kill", "kill-transaction", "show", "top"} {
		c.Assert(slices.Contains(names, name), qt.IsTrue)
	}
}

func TestConnectionsShowHelpListsTargetAndFilterFlags(t *testing.T) {
	c := qt.New(t)

	help := connectionsHelpForTest(c, "show", "--help")
	c.Assert(help, qt.Contains, "--keyspace")
	c.Assert(help, qt.Contains, "--shard")
	c.Assert(help, qt.Contains, "--instance")
	c.Assert(help, qt.Contains, "--role")
}

func TestConnectionsShowHelpKeepsAgentWorkflow(t *testing.T) {
	c := qt.New(t)

	help := connectionsHelpForTest(c, "show", "--help")
	c.Assert(help, qt.Contains, "Use --format json when an agent or script needs to inspect query_id,")
	c.Assert(help, qt.Contains, "transaction_id, and connection_id fields.")
	c.Assert(help, qt.Contains, "Human output uses vertical records so")
	c.Assert(help, qt.Contains, "query text and action IDs are not truncated.")
}

func TestConnectionsKillHelpWarnsAboutDestructiveActions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "kill",
			args: []string{"kill", "--help"},
			want: []string{"destructive", "connection_id", "query_id", "--query"},
		},
		{
			name: "kill transaction",
			args: []string{"kill-transaction", "--help"},
			want: []string{"Postgres", "destructive", "transaction_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			help := connectionsHelpForTest(c, tt.args...)
			for _, want := range tt.want {
				c.Assert(help, qt.Contains, want)
			}
		})
	}
}

func TestBranchCmdRegistersConnectionsAndHiddenProcesslist(t *testing.T) {
	c := qt.New(t)

	cmd := BranchCmd(connectionsTestHelper("acme", ps.DatabaseEngineMySQL, nil, "http://example.invalid", printer.JSON, &bytes.Buffer{}))

	connections := findCommand(cmd, "connections")
	c.Assert(connections, qt.Not(qt.IsNil))
	c.Assert(connections.Hidden, qt.Equals, false)

	processlist := findCommand(cmd, "processlist")
	c.Assert(processlist, qt.Not(qt.IsNil))
	c.Assert(processlist.Hidden, qt.Equals, true)
}

func TestProcesslistHelpPointsToConnectionsCommands(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []string
		wantNot []string
	}{
		{
			name: "show",
			args: []string{"--org", "acme", "processlist", "show", "--help"},
			want: []string{
				"pscale branch connections show",
				"pscale branch connections kill",
				"connection_id",
				"query_id",
			},
			wantNot: []string{"pscale branch processlist kill"},
		},
		{
			name: "kill",
			args: []string{"--org", "acme", "processlist", "kill", "--help"},
			want: []string{
				"pscale branch connections kill",
				"connection_id",
				"query_id",
			},
			wantNot: []string{"as shown in \"pscale branch processlist show\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			help := branchHelpForTest(c, tt.args...)
			for _, want := range tt.want {
				c.Assert(help, qt.Contains, want)
			}
			for _, wantNot := range tt.wantNot {
				c.Assert(help, qt.Not(qt.Contains), wantNot)
			}
		})
	}
}

func TestProcesslistShowMatchesConnectionsShowOutput(t *testing.T) {
	c := qt.New(t)

	body := processlistListBody("commerce", "-80", "zone1-1001", `{"pid":101,"instance":"zone1-1001","usename":"vt_app","client_addr":"10.0.0.12:54231","datname":"checkout","state":"Query","duration_ms":42000,"connection_id":"zone1-1001-101","query_id":"zone1-1001-101","query_text":"SELECT 1"}`)

	var connectionsOut bytes.Buffer
	connectionsServer := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/acme/databases/shop/branches/main/connections")
		assertQueryParam(c, r, "keyspace", "commerce")
		assertQueryParam(c, r, "shard", "-80")
		_, _ = io.WriteString(w, body)
	})
	connections := BranchCmd(connectionsTestHelper("acme", ps.DatabaseEngineMySQL, nil, connectionsServer.URL, printer.JSON, &connectionsOut))
	connections.SetArgs([]string{"--org", "acme", "connections", "show", "shop", "main", "--keyspace", "commerce", "--shard", "-80"})

	var processlistOut bytes.Buffer
	processlistServer := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/acme/databases/shop/branches/main/connections")
		assertQueryParam(c, r, "keyspace", "commerce")
		assertQueryParam(c, r, "shard", "-80")
		_, _ = io.WriteString(w, body)
	})
	processlist := BranchCmd(connectionsTestHelper("acme", ps.DatabaseEngineMySQL, nil, processlistServer.URL, printer.JSON, &processlistOut))
	processlist.SetArgs([]string{"--org", "acme", "processlist", "show", "shop", "main", "--keyspace", "commerce", "--shard", "-80"})

	c.Assert(connections.Execute(), qt.IsNil)
	c.Assert(processlist.Execute(), qt.IsNil)
	assertJSONEqual(c, connectionsOut.String(), processlistOut.String())
}

func TestProcesslistRejectsPostgresBranches(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "show", args: []string{"--org", "acme", "processlist", "show", "pgload", "main"}},
		{name: "kill", args: []string{"--org", "acme", "processlist", "kill", "pgload", "main", "101"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
				c.Fatalf("processlist should reject Postgres before calling %s", r.URL.Path)
			})

			out, errOut, err := executeBranchCommandForTest(
				connectionsTestHelper("acme", ps.DatabaseEnginePostgres, nil, server.URL, printer.JSON, &bytes.Buffer{}),
				tt.args,
			)

			c.Assert(err, qt.ErrorMatches, "processlist is only supported for Vitess databases")
			c.Assert(out, qt.Equals, "")
			c.Assert(errOut, qt.Equals, "")
		})
	}
}

func TestConnectionsShowDispatchesVitess(t *testing.T) {
	c := qt.New(t)

	server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/acme/databases/shop/branches/main/connections")
		assertQueryParam(c, r, "keyspace", "commerce")
		assertQueryParam(c, r, "shard", "-80")
		_, _ = io.WriteString(w, processlistListBody("commerce", "-80", "zone1-1001", `{"pid":101,"instance":"zone1-1001","usename":"vt_app","client_addr":"10.0.0.12:54231","datname":"checkout","state":"Query","duration_ms":42000,"connection_id":"zone1-1001-101","query_id":"zone1-1001-101","query_text":"SELECT 1"}`))
	})

	var out bytes.Buffer
	cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEngineMySQL, nil, server.URL, printer.JSON, &out))
	cmd.SetArgs([]string{"show", "shop", "main", "--keyspace", "commerce", "--shard", "-80"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(out.String(), qt.Contains, `"topology": {`)
	c.Assert(out.String(), qt.Contains, `"tablet": "zone1-1001"`)
	c.Assert(out.String(), qt.Contains, `"database": "checkout"`)
	c.Assert(out.String(), qt.Contains, `"connection_id": "zone1-1001-101"`)
}

func TestConnectionsShowRejectsEngineSpecificFlags(t *testing.T) {
	tests := []struct {
		name    string
		engine  ps.DatabaseEngine
		args    []string
		wantErr string
	}{
		{
			name:    "vitess rejects postgres instance filter",
			engine:  ps.DatabaseEngineMySQL,
			args:    []string{"show", "shop", "main", "--instance", "primary"},
			wantErr: "--instance/--role are only supported for Postgres databases",
		},
		{
			name:    "vitess rejects postgres role filter",
			engine:  ps.DatabaseEngineMySQL,
			args:    []string{"show", "shop", "main", "--role", "primary"},
			wantErr: "--instance/--role are only supported for Postgres databases",
		},
		{
			name:    "postgres rejects vitess target",
			engine:  ps.DatabaseEnginePostgres,
			args:    []string{"show", "pgload", "main", "--keyspace", "commerce", "--shard", "-80"},
			wantErr: "--keyspace/--shard are only supported for Vitess databases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
				c.Fatalf("show should reject flags before calling %s", r.URL.Path)
			})
			cmd := connectionsCmdForTest(connectionsTestHelper("acme", tt.engine, nil, server.URL, printer.JSON, &bytes.Buffer{}))
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			c.Assert(err, qt.ErrorMatches, tt.wantErr)
		})
	}
}

func TestEngineFlagValidationAfterLookupFailure(t *testing.T) {
	c := qt.New(t)

	server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
		c.Fatalf("show should surface database lookup errors before calling %s", r.URL.Path)
	})
	databases := &mock.DatabaseService{
		GetFn: func(context.Context, *ps.GetDatabaseRequest) (*ps.Database, error) {
			return nil, errors.New("database lookup failed")
		},
	}
	cmd := connectionsCmdForTest(connectionsTestHelperWithDatabaseService("acme", databases, nil, server.URL, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"show", "shop", "main", "--keyspace", "commerce", "--instance", "primary"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "database lookup failed")
}

func TestConnectionsShowDispatchesPostgres(t *testing.T) {
	c := qt.New(t)

	var gotPath string
	server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		_, _ = io.WriteString(w, sampleBranchConnectionsListResponse())
	})

	var out bytes.Buffer
	cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEnginePostgres, nil, server.URL, printer.JSON, &out))
	cmd.SetArgs([]string{"show", "pgload", "main", "--instance", "primary"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(gotPath, qt.Equals, "/v1/organizations/acme/databases/pgload/branches/main/connections")
	c.Assert(out.String(), qt.Contains, `"connection_id": "primary-123-c"`)
	c.Assert(out.String(), qt.Not(qt.Contains), `"topology"`)
}

func TestConnectionsKillDispatchesVitess(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantKind string
		wantID   string
	}{
		{
			name:     "connection",
			args:     []string{"kill", "shop", "main", "zone1-1001-101", "--keyspace", "commerce", "--shard", "-80"},
			wantKind: "connection",
			wantID:   "zone1-1001-101",
		},
		{
			name:     "query",
			args:     []string{"kill", "shop", "main", "zone1-1001-101", "--query", "--keyspace", "commerce", "--shard", "-80"},
			wantKind: "query",
			wantID:   "zone1-1001-101",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			wantPath := "/v1/organizations/acme/databases/shop/branches/main/connections/connection/" + tt.wantID
			if tt.wantKind == "query" {
				wantPath = "/v1/organizations/acme/databases/shop/branches/main/connections/query/" + tt.wantID
			}
			server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
				c.Assert(r.Method, qt.Equals, http.MethodDelete)
				c.Assert(r.URL.Path, qt.Equals, wantPath)
				assertQueryParam(c, r, "keyspace", "commerce")
				assertQueryParam(c, r, "shard", "-80")
				_, _ = io.WriteString(w, `{"success":true,"id":101,"kind":"`+tt.wantKind+`","keyspace":"commerce","shard":"-80","tablet":"zone1-1001"}`)
			})

			var out bytes.Buffer
			cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEngineMySQL, nil, server.URL, printer.JSON, &out))
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			c.Assert(err, qt.IsNil)
		})
	}
}

func TestConnectionsKillRejectsWrongEngineTargetFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "connection", args: []string{"kill", "pgload", "main", "primary-123-c", "--keyspace", "commerce", "--shard", "-80"}},
		{name: "query", args: []string{"kill", "pgload", "main", "primary-123-q", "--query", "--keyspace", "commerce", "--shard", "-80"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
				c.Fatalf("kill should reject flags before calling %s", r.URL.Path)
			})
			cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEnginePostgres, nil, server.URL, printer.JSON, &bytes.Buffer{}))
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			c.Assert(err, qt.ErrorMatches, "--keyspace/--shard are only supported for Vitess databases")
		})
	}
}

func TestProcesslistKillKeepsNumericIDValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "not numeric", args: []string{"--org", "acme", "processlist", "kill", "shop", "main", "primary-123-c"}},
		{name: "zero", args: []string{"--org", "acme", "processlist", "kill", "shop", "main", "0"}},
		{name: "negative", args: []string{"--org", "acme", "processlist", "kill", "shop", "main", "--", "-1"}},
		{name: "negative without separator", args: []string{"--org", "acme", "processlist", "kill", "shop", "main", "-1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			out, errOut, err := executeBranchCommandForTest(
				processlistTestHelper("acme", "http://127.0.0.1:1", printer.JSON, &bytes.Buffer{}),
				tt.args,
			)

			c.Assert(err, qt.ErrorMatches, "id must be a positive integer")
			c.Assert(out, qt.Equals, "")
			c.Assert(errOut, qt.Equals, "")
		})
	}
}

func TestConnectionsKillDispatchesPostgres(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
	}{
		{name: "connection", args: []string{"kill", "pgload", "main", "primary-123-c"}, wantPath: "/v1/organizations/acme/databases/pgload/branches/main/connections/connection/primary-123-c"},
		{name: "query", args: []string{"kill", "pgload", "main", "primary-123-q", "--query"}, wantPath: "/v1/organizations/acme/databases/pgload/branches/main/connections/query/primary-123-q"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
				c.Assert(r.Method, qt.Equals, http.MethodDelete)
				c.Assert(r.URL.Path, qt.Equals, tt.wantPath)
				w.WriteHeader(http.StatusNoContent)
			})

			cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEnginePostgres, nil, server.URL, printer.JSON, &bytes.Buffer{}))
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			c.Assert(err, qt.IsNil)
		})
	}
}

func TestConnectionsKillTrimsPostgresIDAndRejectsEmpty(t *testing.T) {
	c := qt.New(t)

	var gotPath string
	server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})

	cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEnginePostgres, nil, server.URL, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"kill", "pgload", "main", " primary-123-c "})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(gotPath, qt.Equals, "/v1/organizations/acme/databases/pgload/branches/main/connections/connection/primary-123-c")

	empty := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEnginePostgres, nil, server.URL, printer.JSON, &bytes.Buffer{}))
	empty.SetArgs([]string{"kill", "pgload", "main", " "})
	err = empty.Execute()
	c.Assert(err, qt.ErrorMatches, "connection-id is required")
}

func TestConnectionsKillRejectsEmptyIDBeforeLookup(t *testing.T) {
	c := qt.New(t)

	databases := &mock.DatabaseService{
		GetFn: func(context.Context, *ps.GetDatabaseRequest) (*ps.Database, error) {
			return nil, errors.New("database lookup should not be called")
		},
	}
	cmd := connectionsCmdForTest(connectionsTestHelperWithDatabaseService("acme", databases, nil, "http://127.0.0.1:1", printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"kill", "pgload", "main", " "})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "connection-id is required")
	c.Assert(databases.GetFnInvoked, qt.IsFalse)
}

func TestConnectionsKillTransactionDispatches(t *testing.T) {
	c := qt.New(t)

	server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/acme/databases/pgload/branches/main/connections/transaction/primary-123-t")
		w.WriteHeader(http.StatusNoContent)
	})

	pg := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEnginePostgres, nil, server.URL, printer.JSON, &bytes.Buffer{}))
	pg.SetArgs([]string{"kill-transaction", "pgload", "main", "primary-123-t"})
	c.Assert(pg.Execute(), qt.IsNil)
}

func TestConnectionsKillTransactionRejectsEmptyIDBeforeLookup(t *testing.T) {
	c := qt.New(t)

	databases := &mock.DatabaseService{
		GetFn: func(context.Context, *ps.GetDatabaseRequest) (*ps.Database, error) {
			return nil, errors.New("database lookup should not be called")
		},
	}
	cmd := connectionsCmdForTest(connectionsTestHelperWithDatabaseService("acme", databases, nil, "http://127.0.0.1:1", printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"kill-transaction", "pgload", "main", " "})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "transaction-id is required")
	c.Assert(databases.GetFnInvoked, qt.IsFalse)
}

func TestConnectionsKillTransactionRejectsVitess(t *testing.T) {
	c := qt.New(t)

	server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
		c.Fatalf("kill-transaction should reject Vitess before calling %s", r.URL.Path)
	})
	cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEngineMySQL, nil, server.URL, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"kill-transaction", "shop", "main", "tx-123"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "connections kill-transaction is only supported for Postgres databases")
}

func TestPostgresActionResultOmitsVitessTopologyColumns(t *testing.T) {
	c := qt.New(t)

	server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/acme/databases/pgload/branches/main/connections/connection/primary-123-c")
		_, _ = io.WriteString(w, `{"success":true,"id":101,"kind":"connection","keyspace":"commerce","shard":"-80","tablet":"zone1-1001"}`)
	})
	var out bytes.Buffer
	cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEnginePostgres, nil, server.URL, printer.CSV, &out))
	cmd.SetArgs([]string{"kill", "pgload", "main", "primary-123-c"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	headers := readCSVRows(c, out.String())[0]
	c.Assert(headers, qt.Not(qt.Contains), "keyspace")
	c.Assert(headers, qt.Not(qt.Contains), "shard")
	c.Assert(headers, qt.Not(qt.Contains), "tablet")
	c.Assert(headers, qt.Contains, "success")
	c.Assert(headers, qt.Contains, "id")
	c.Assert(headers, qt.Contains, "kind")
}

func TestVitessActionResultKeepsTopologyShape(t *testing.T) {
	c := qt.New(t)

	server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/acme/databases/shop/branches/main/connections/connection/zone1-1001-101")
		assertQueryParam(c, r, "keyspace", "commerce")
		assertQueryParam(c, r, "shard", "-80")
		_, _ = io.WriteString(w, `{"success":true,"id":101,"kind":"connection","keyspace":"commerce","shard":"-80","tablet":"zone1-1001"}`)
	})
	var out bytes.Buffer
	cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEngineMySQL, nil, server.URL, printer.CSV, &out))
	cmd.SetArgs([]string{"kill", "shop", "main", "zone1-1001-101", "--keyspace", "commerce", "--shard", "-80"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	rows := readCSVRows(c, out.String())
	c.Assert(rows[0], qt.Contains, "keyspace")
	c.Assert(rows[0], qt.Contains, "shard")
	c.Assert(rows[0], qt.Contains, "tablet")
	c.Assert(rows[1], qt.Contains, "commerce")
	c.Assert(rows[1], qt.Contains, "-80")
	c.Assert(rows[1], qt.Contains, "zone1-1001")
}

func TestActionResultJSONShapeStable(t *testing.T) {
	c := qt.New(t)

	server := liveConnectionsBranchServer(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		_, _ = io.WriteString(w, `{"success":true,"id":101,"kind":"connection","keyspace":"commerce","shard":"-80","tablet":"zone1-1001"}`)
	})
	var out bytes.Buffer
	cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEnginePostgres, nil, server.URL, printer.JSON, &out))
	cmd.SetArgs([]string{"kill", "pgload", "main", "primary-123-c"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	var got map[string]any
	c.Assert(json.Unmarshal(out.Bytes(), &got), qt.IsNil)
	c.Assert(got, qt.DeepEquals, map[string]any{
		"success":  true,
		"keyspace": "commerce",
		"shard":    "-80",
		"tablet":   "zone1-1001",
		"id":       float64(101),
		"kind":     "connection",
	})
}

func TestConnectionsRoleValidationStillWorksOnPostgres(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "unknown role", args: []string{"show", "pgload", "main", "--role", "writer"}, wantErr: "--role must be primary or replica"},
		{name: "role with instance", args: []string{"show", "pgload", "main", "--role", "primary", "--instance", "primary"}, wantErr: "--role cannot be combined with --instance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEnginePostgres, nil, "http://127.0.0.1:1", printer.JSON, &bytes.Buffer{}))
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			c.Assert(err, qt.ErrorMatches, tt.wantErr)
		})
	}
}

func TestConnectionsShowRejectsInvalidRoleBeforeLookup(t *testing.T) {
	c := qt.New(t)

	databases := &mock.DatabaseService{
		GetFn: func(context.Context, *ps.GetDatabaseRequest) (*ps.Database, error) {
			return nil, errors.New("database lookup should not be called")
		},
	}
	cmd := connectionsCmdForTest(connectionsTestHelperWithDatabaseService("acme", databases, nil, "http://127.0.0.1:1", printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"show", "pgload", "main", "--role", "writer"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "--role must be primary or replica")
	c.Assert(databases.GetFnInvoked, qt.IsFalse)
}

func connectionsTestHelper(org string, engine ps.DatabaseEngine, processlist ps.ProcesslistService, baseURL string, format printer.Format, out *bytes.Buffer) *cmdutil.Helper {
	return connectionsTestHelperWithDatabaseService(org, databaseServiceForEngine(org, engine), processlist, baseURL, format, out)
}

func connectionsTestHelperWithDatabaseService(org string, databases ps.DatabasesService, processlist ps.ProcesslistService, baseURL string, format printer.Format, out *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(out)
	p.SetResourceOutput(out)

	return &cmdutil.Helper{
		Config: &config.Config{
			AccessToken:  "token",
			BaseURL:      baseURL,
			Organization: org,
		},
		Printer: p,
		Client: func() (*ps.Client, error) {
			return &ps.Client{Databases: databases, Processlist: processlist}, nil
		},
	}
}

func databaseServiceForEngine(org string, engine ps.DatabaseEngine) *mock.DatabaseService {
	return &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			if req.Organization != org {
				return nil, errors.New("unexpected organization")
			}
			return &ps.Database{Name: req.Database, Kind: engine}, nil
		},
	}
}

func connectionsCmdForTest(ch *cmdutil.Helper) *cobra.Command {
	cmd := ConnectionsCmd(ch)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetErr(io.Discard)
	return cmd
}

func connectionsHelpForTest(c *qt.C, args ...string) string {
	var out bytes.Buffer
	cmd := connectionsCmdForTest(connectionsTestHelper("acme", ps.DatabaseEngineMySQL, nil, "http://example.invalid", printer.Human, &out))
	cmd.SetOut(&out)
	cmd.SetArgs(args)

	c.Assert(cmd.Execute(), qt.IsNil)
	return out.String()
}

func branchHelpForTest(c *qt.C, args ...string) string {
	var out bytes.Buffer
	cmd := BranchCmd(connectionsTestHelper("acme", ps.DatabaseEngineMySQL, nil, "http://example.invalid", printer.Human, &out))
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetOut(&out)
	cmd.SetArgs(args)

	c.Assert(cmd.Execute(), qt.IsNil)
	return out.String()
}

func assertQueryParam(c *qt.C, r *http.Request, key, want string) {
	c.Assert(r.URL.Query().Get(key), qt.Equals, want)
}

func assertJSONEqual(c *qt.C, want, got string) {
	var wantJSON any
	var gotJSON any
	c.Assert(json.Unmarshal([]byte(want), &wantJSON), qt.IsNil)
	c.Assert(json.Unmarshal([]byte(got), &gotJSON), qt.IsNil)
	c.Assert(gotJSON, qt.DeepEquals, wantJSON)
}

func liveConnectionsBranchServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func sampleBranchConnectionsListResponse() string {
	return `{"type":"list","database_kind":"postgresql","captured_at":"2026-04-29T12:34:56Z","instances":[{"id":"primary","role":"primary","error":null}],"data":[{"pid":123,"instance":"primary","duration_ms":664000,"state":"active","usename":"alice","application_name":"psql","client_addr":"10.0.0.1","query_text":"SELECT pg_sleep(600)","xact_start":"2026-04-29T12:23:52Z","query_start":"2026-04-29T12:23:52Z","query_id":"primary-123-q","transaction_id":"primary-123-t","connection_id":"primary-123-c"}]}`
}

func commandNames(cmd *cobra.Command) []string {
	names := make([]string, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		names = append(names, child.Name())
	}
	return names
}

func findCommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, child := range cmd.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}

func executeBranchCommandForTest(ch *cmdutil.Helper, args []string) (string, string, error) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := BranchCmd(ch)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)

	err := cmd.Execute()

	return out.String(), errOut.String(), err
}

func readCSVRows(c *qt.C, raw string) [][]string {
	rows, err := csv.NewReader(strings.NewReader(raw)).ReadAll()
	c.Assert(err, qt.IsNil)
	return rows
}
