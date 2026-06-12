package connections

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
)

func TestActionRunnersIssueDeletesWithExplicitIDs(t *testing.T) {
	tests := []struct {
		name      string
		run       func(context.Context, *cmdutil.Helper, string, string, string, ConnectionTarget) error
		id        string
		wantPath  string
		wantQuery string
		target    ConnectionTarget
	}{
		{
			name:     "cancel query",
			run:      RunCancelQuery,
			id:       "primary-123-q",
			wantPath: "/v1/organizations/acme/databases/pgload/branches/main/connections/query/primary-123-q",
		},
		{
			name:     "kill transaction",
			run:      RunKillTransaction,
			id:       "primary-123-t",
			wantPath: "/v1/organizations/acme/databases/pgload/branches/main/connections/transaction/primary-123-t",
		},
		{
			name:      "kill connection with target",
			run:       RunKillConnection,
			id:        "zone1-1001-101",
			wantPath:  "/v1/organizations/acme/databases/pgload/branches/main/connections/connection/zone1-1001-101",
			wantQuery: "keyspace=commerce&shard=-80",
			target:    ConnectionTarget{Keyspace: "commerce", Shard: "-80"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Assert(r.Method, qt.Equals, http.MethodDelete)
				c.Assert(r.URL.Path, qt.Equals, tt.wantPath)
				c.Assert(r.URL.RawQuery, qt.Equals, tt.wantQuery)
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(server.Close)

			var out bytes.Buffer
			ch := connectionsTestHelper(server.URL, printer.Human, &out)
			err := tt.run(context.Background(), ch, "pgload", "main", tt.id, tt.target)

			c.Assert(err, qt.IsNil)
		})
	}
}

func TestActionRunnersPrintResults(t *testing.T) {
	tests := []struct {
		name     string
		format   printer.Format
		run      func(context.Context, *cmdutil.Helper, string, string, string, ConnectionTarget) error
		response string
		want     []string
	}{
		{
			name:     "json",
			format:   printer.JSON,
			run:      RunCancelQuery,
			response: `{"success":true,"keyspace":"commerce","shard":"-80","tablet":"zone1-1001","id":101,"kind":"query"}`,
			want:     []string{`"success": true`, `"keyspace": "commerce"`, `"kind": "query"`},
		},
		{
			name:     "transaction",
			format:   printer.JSON,
			run:      RunKillTransaction,
			response: `{"success":true,"id":101,"kind":"transaction"}`,
			want:     []string{`"success": true`, `"kind": "transaction"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tt.response)
			}))
			t.Cleanup(server.Close)

			var out bytes.Buffer
			ch := connectionsTestHelper(server.URL, tt.format, &out)
			err := tt.run(context.Background(), ch, "pgload", "main", "zone1-1001-101", ConnectionTarget{Keyspace: "commerce", Shard: "-80"})

			c.Assert(err, qt.IsNil)
			for _, want := range tt.want {
				c.Assert(out.String(), qt.Contains, want)
			}
		})
	}
}

func TestActionRunnersPrintSentenceInHumanFormat(t *testing.T) {
	tests := []struct {
		name     string
		run      func(context.Context, *cmdutil.Helper, string, string, string, ConnectionTarget) error
		response string
		want     string
	}{
		{
			name:     "cancel query",
			run:      RunCancelQuery,
			response: `{"success":true}`,
			want:     "Cancelled query.\n",
		},
		{
			name:     "kill transaction",
			run:      RunKillTransaction,
			response: `{"success":true}`,
			want:     "Killed transaction.\n",
		},
		{
			name:     "kill connection",
			run:      RunKillConnection,
			response: `{"success":true,"tablet":"zone1-1001"}`,
			want:     "Killed connection on zone1-1001.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tt.response)
			}))
			t.Cleanup(server.Close)

			var out bytes.Buffer
			ch := connectionsTestHelper(server.URL, printer.Human, &out)
			err := tt.run(context.Background(), ch, "pgload", "main", "id-1", ConnectionTarget{})

			c.Assert(err, qt.IsNil)
			c.Assert(out.String(), qt.Equals, tt.want)
		})
	}
}

func TestActionRunnersRejectMissingID(t *testing.T) {
	tests := []struct {
		name    string
		run     func(context.Context, *cmdutil.Helper, string, string, string, ConnectionTarget) error
		wantErr string
	}{
		{name: "cancel query", run: RunCancelQuery, wantErr: "query-id is required"},
		{name: "kill transaction", run: RunKillTransaction, wantErr: "transaction-id is required"},
		{name: "kill connection", run: RunKillConnection, wantErr: "connection-id is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			var out bytes.Buffer
			ch := connectionsTestHelper("http://example.invalid", printer.Human, &out)

			err := tt.run(context.Background(), ch, "pgload", "main", " ", ConnectionTarget{})

			c.Assert(err, qt.ErrorMatches, tt.wantErr)
		})
	}
}

func TestActionRunnersSurfaceServerErrors(t *testing.T) {
	tests := []struct {
		name       string
		run        func(context.Context, *cmdutil.Helper, string, string, string, ConnectionTarget) error
		id         string
		statusCode int
		body       string
		wantErr    string
	}{
		{
			name:       "kill transaction",
			run:        RunKillTransaction,
			id:         "primary-123-t",
			statusCode: http.StatusUnprocessableEntity,
			body:       `{"code":"verification_mismatch","message":"connection no longer matches the expected snapshot"}`,
			wantErr:    "terminate transaction: connection no longer matches the expected snapshot",
		},
		{
			name:       "cancel query",
			run:        RunCancelQuery,
			id:         "primary-123-q",
			statusCode: http.StatusNotFound,
			wantErr:    "cancel query: query_id not found.*",
		},
		{
			name:       "kill connection",
			run:        RunKillConnection,
			id:         "primary-123-c",
			statusCode: http.StatusNotFound,
			wantErr:    "terminate connection: connection_id not found.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = io.WriteString(w, tt.body)
			}))
			t.Cleanup(server.Close)

			var out bytes.Buffer
			ch := connectionsTestHelper(server.URL, printer.Human, &out)
			err := tt.run(context.Background(), ch, "pgload", "main", tt.id, ConnectionTarget{})

			c.Assert(err, qt.ErrorMatches, tt.wantErr)
		})
	}
}

func connectionsTestHelper(baseURL string, format printer.Format, out *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(out)
	p.SetResourceOutput(out)

	return &cmdutil.Helper{
		Config: &config.Config{
			BaseURL:        baseURL,
			Organization:   "acme",
			ServiceTokenID: "tid",
			ServiceToken:   "secret",
		},
		Printer: p,
	}
}
