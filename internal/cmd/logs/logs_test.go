package logs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestCreateQueryMatchesAppQueryShape(t *testing.T) {
	tests := []struct {
		name  string
		flags *flags
		want  string
	}{
		{
			name:  "defaults",
			flags: &flags{period: "1h", servers: []string{"primary"}, limit: 100, page: 1},
			want:  "* _time:1h (planetscale.role:primary) | sort by (_time desc) | offset 0",
		},
		{
			name: "filters pagination and pipeline",
			flags: &flags{
				query:   "timeout | unpack_json | fields _msg",
				period:  "6h",
				levels:  []string{"ERROR", "WARNING"},
				servers: []string{"primary", "pod-2"},
				shards:  []string{"shard-1", "shard-2"},
				limit:   25,
				page:    3,
			},
			want: "timeout _time:6h (planetscale.level:ERROR OR planetscale.level:WARNING) " +
				"(planetscale.shard:shard-1 OR planetscale.shard:shard-2) " +
				"(planetscale.role:primary OR planetscale.pod:pod-2) " +
				"| unpack_json | fields _msg | sort by (_time desc) | offset 50",
		},
		{
			name:  "all servers",
			flags: &flags{period: "1h", servers: nil, limit: 100, page: 1},
			want:  "* _time:1h | sort by (_time desc) | offset 0",
		},
		{
			name: "custom time range",
			flags: &flags{
				period:  "1h",
				from:    "2026-08-20T16:00:00Z",
				to:      "2026-08-20T18:00:00Z",
				servers: []string{"primary"},
				limit:   100,
				page:    1,
			},
			want: "* _time:[2026-08-20T16:00:00Z, 2026-08-20T18:00:00Z] " +
				"(planetscale.role:primary) | sort by (_time desc) | offset 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createQuery(tt.flags); got != tt.want {
				t.Fatalf("query = %q\nwant  = %q", got, tt.want)
			}
		})
	}
}

func TestValidateFlags(t *testing.T) {
	f := &flags{period: "1h", levels: []string{"error"}, limit: 100, page: 1}
	if err := validateFlags(f, false); err != nil {
		t.Fatal(err)
	}
	if f.levels[0] != "ERROR" {
		t.Fatalf("normalized level = %q, want ERROR", f.levels[0])
	}

	for _, tt := range []struct {
		name          string
		f             *flags
		periodChanged bool
		want          string
	}{
		{name: "period", f: &flags{period: "2h", limit: 100, page: 1}, want: "invalid --period"},
		{name: "limit", f: &flags{period: "1h", limit: 0, page: 1}, want: "--limit must be greater"},
		{name: "page", f: &flags{period: "1h", limit: 100, page: 0}, want: "--page must be greater"},
		{name: "level", f: &flags{period: "1h", levels: []string{"TRACE"}, limit: 100, page: 1}, want: "invalid --level"},
		{name: "from without to", f: &flags{period: "1h", from: "2026-08-20T16:00:00Z", limit: 100, page: 1}, want: "--from and --to must be used together"},
		{name: "to without from", f: &flags{period: "1h", to: "2026-08-20T18:00:00Z", limit: 100, page: 1}, want: "--from and --to must be used together"},
		{
			name:          "range with explicit period",
			f:             &flags{period: "1h", from: "2026-08-20T16:00:00Z", to: "2026-08-20T18:00:00Z", limit: 100, page: 1},
			periodChanged: true,
			want:          "--period cannot be combined with --from and --to",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFlags(tt.f, tt.periodChanged)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestParseLogsMatchesAppBehavior(t *testing.T) {
	raw := strings.Join([]string{
		`{"_time":"2026-08-21T12:00:00Z","_stream_id":"stream-1","_msg":"{\"message\":\"database started\"}","planetscale.level":"DEBUG","planetscale.role":"primary","planetscale.shard":"s1","planetscale.container":"postgres","planetscale.availability_zone":"us-east-1a","planetscale.pod":"pod-1"}`,
		`{"_time":"2026-08-21T12:01:00Z","_stream_id":"stream-2","_msg":"plain message","planetscale.container":"postgres","planetscale.availability_zone":"us-east-1b","planetscale.pod":"pod-2"}`,
		`not json`,
		`{"_msg":"{\"other\":\"missing message\"}"}`,
	}, "\n")

	entries, err := parseLogs(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Message != "database started" || entries[0].RawMessage != `{"message":"database started"}` {
		t.Fatalf("unexpected nested entry: %#v", entries[0])
	}
	if entries[1].Message != "plain message" || entries[1].Level != "INFO" {
		t.Fatalf("unexpected plain entry: %#v", entries[1])
	}
}

func TestPrintHumanLogsLineOriented(t *testing.T) {
	format := printer.Human
	var out bytes.Buffer
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	ch := &cmdutil.Helper{Printer: p}

	printHumanLogs(ch, []*LogEntry{
		{Time: "2026-08-21T12:00:00Z", Level: "ERROR", Message: "connection refused"},
		{Time: "2026-08-21T12:00:01Z", Level: "INFO", Message: "first line\nsecond\tcolumn"},
	})

	want := "2026-08-21T12:00:00Z ERROR   connection refused\n" +
		`2026-08-21T12:00:01Z INFO    first line\nsecond\tcolumn` + "\n"
	if got := out.String(); got != want {
		t.Fatalf("human output = %q, want %q", got, want)
	}
}

func TestLogsCmdPrinterFormatsAndSignedQuery(t *testing.T) {
	for _, format := range []printer.Format{printer.Human, printer.JSON, printer.CSV} {
		t.Run(format.String(), func(t *testing.T) {
			wantLogsQuery := "failed _time:6h (planetscale.level:ERROR OR planetscale.level:WARNING) " +
				"(planetscale.shard:s1 OR planetscale.shard:s2) " +
				"(planetscale.role:primary OR planetscale.pod:pod-2) " +
				"| fields _msg | sort by (_time desc) | offset 50"
			args := []string{"db", "main", "--org", "acme", "--query", "failed | fields _msg", "--period", "6h", "--level", "error,warning", "--server", "primary,pod-2", "--shard", "s1,s2", "--limit", "25", "--page", "3"}
			if format == printer.JSON {
				wantLogsQuery = "failed _time:[2026-08-20T16:00:00Z, 2026-08-20T18:00:00Z] " +
					"(planetscale.level:ERROR OR planetscale.level:WARNING) " +
					"(planetscale.shard:s1 OR planetscale.shard:s2) " +
					"(planetscale.role:primary OR planetscale.pod:pod-2) " +
					"| fields _msg | sort by (_time desc) | offset 50"
				args = []string{"db", "main", "--org", "acme", "--query", "failed | fields _msg", "--from", "2026-08-20T16:00:00Z", "--to", "2026-08-20T18:00:00Z", "--level", "error,warning", "--server", "primary,pod-2", "--shard", "s1,s2", "--limit", "25", "--page", "3"}
			}

			var serverURL string
			mux := http.NewServeMux()
			mux.HandleFunc("/v1/organizations/acme/databases/db/branches/main/logs/signatures", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method = %s, want POST", r.Method)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("authorization = %q, want Bearer token", got)
				}
				_ = json.NewEncoder(w).Encode(map[string]string{
					"sig": "signature",
					"exp": "999",
					"url": serverURL + "/signed?sig=signature&exp=999",
				})
			})
			mux.HandleFunc("/signed", func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "" {
					t.Fatalf("logs request leaked authorization %q", got)
				}
				assertSignedQuery(t, r.URL.Query(), wantLogsQuery)
				_, _ = io.WriteString(w, strings.Join([]string{
					`{"_time":"2026-08-21T12:00:00Z","_stream_id":"stream-1","_msg":"{\"message\":\"database started\"}","planetscale.level":"ERROR","planetscale.role":"primary","planetscale.shard":"s1","planetscale.container":"postgres","planetscale.availability_zone":"us-east-1a","planetscale.pod":"pod-1"}`,
					`malformed`,
				}, "\n"))
			})

			server := httptest.NewServer(mux)
			serverURL = server.URL
			t.Cleanup(server.Close)

			var out bytes.Buffer
			ch := logsTestHelper(server.URL, format, &out)
			cmd := LogsCmd(ch)
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}

			switch format {
			case printer.JSON:
				var entries []*LogEntry
				if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
					t.Fatalf("invalid JSON output %q: %v", out.String(), err)
				}
				if len(entries) != 1 || entries[0].Message != "database started" || entries[0].Level != "ERROR" {
					t.Fatalf("unexpected JSON entries: %#v", entries)
				}
			case printer.CSV:
				if !strings.Contains(out.String(), "time,level,message") || !strings.Contains(out.String(), "database started") {
					t.Fatalf("unexpected CSV output: %q", out.String())
				}
			case printer.Human:
				want := "2026-08-21T12:00:00Z ERROR   database started\n"
				if got := out.String(); got != want {
					t.Fatalf("human output = %q, want %q", got, want)
				}
			}
		})
	}
}

func TestLogsCmdSignedEndpointFailure(t *testing.T) {
	var serverURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/organizations/acme/databases/db/branches/main/logs/signatures", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"url": serverURL + "/signed"})
	})
	mux.HandleFunc("/signed", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprint(w, "logs are warming up")
	})

	server := httptest.NewServer(mux)
	serverURL = server.URL
	t.Cleanup(server.Close)

	var out bytes.Buffer
	cmd := LogsCmd(logsTestHelper(server.URL, printer.JSON, &out))
	cmd.SetArgs([]string{"db", "main", "--org", "acme"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "HTTP 503 Service Unavailable: logs are warming up") {
		t.Fatalf("error = %v", err)
	}
}

func assertSignedQuery(t *testing.T, query url.Values, wantLogsQuery string) {
	t.Helper()
	for key, want := range map[string]string{
		"sig":   "signature",
		"exp":   "999",
		"limit": "25",
		"query": wantLogsQuery,
	} {
		if got := query.Get(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func logsTestHelper(baseURL string, format printer.Format, out *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(out)
	p.SetHumanOutput(out)

	return &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			AccessToken:  "token",
			Organization: "acme",
			BaseURL:      baseURL,
		},
		Client: func() (*ps.Client, error) {
			return ps.NewClient(ps.WithBaseURL(baseURL), ps.WithAccessToken("token"))
		},
	}
}
