package connections

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func TestFilterConnectionList(t *testing.T) {
	tests := []struct {
		name          string
		input         live.ConnectionList
		filter        connectionFilter
		wantPIDs      []int
		wantInstances []live.InstanceMeta
	}{
		{
			name: "by instance",
			input: live.ConnectionList{Connections: []live.Connection{
				{PID: 1, Instance: "primary", InstanceRole: "primary"},
				{PID: 2, Instance: "replica-a", InstanceRole: "replica"},
				{PID: 3, Instance: "replica-b", InstanceRole: "replica"},
			}},
			filter:   connectionFilter{instance: "replica-a"},
			wantPIDs: []int{2},
		},
		{
			name: "by primary role",
			input: live.ConnectionList{Connections: []live.Connection{
				{PID: 1, Instance: "primary", InstanceRole: "primary"},
				{PID: 2, Instance: "replica-a", InstanceRole: "replica"},
			}},
			filter:   connectionFilter{role: "primary"},
			wantPIDs: []int{1},
		},
		{
			name: "by replica role",
			input: live.ConnectionList{Connections: []live.Connection{
				{PID: 1, Instance: "primary", InstanceRole: "primary"},
				{PID: 2, Instance: "replica-a", InstanceRole: "replica"},
				{PID: 3, Instance: "replica-b", InstanceRole: "replica"},
			}},
			filter:   connectionFilter{role: "replica"},
			wantPIDs: []int{2, 3},
		},
		{
			name: "no filter passes through",
			input: live.ConnectionList{Connections: []live.Connection{
				{PID: 1, Instance: "primary", InstanceRole: "primary"},
			}},
			filter:   connectionFilter{},
			wantPIDs: []int{1},
		},
		{
			name: "role filters instances metadata",
			input: live.ConnectionList{
				Connections: []live.Connection{
					{PID: 1, Instance: "primary", InstanceRole: "primary"},
				},
				Instances: []live.InstanceMeta{
					{ID: "primary", Role: "primary"},
					{ID: "replica-1", Role: "replica"},
					{ID: "replica-2", Role: "replica", Error: "timeout"},
				},
			},
			filter:   connectionFilter{role: "primary"},
			wantPIDs: []int{1},
			wantInstances: []live.InstanceMeta{
				{ID: "primary", Role: "primary"},
			},
		},
		{
			name: "instance filter scopes instances to target",
			input: live.ConnectionList{
				Instances: []live.InstanceMeta{
					{ID: "primary", Role: "primary"},
					{ID: "replica-1", Role: "replica"},
					{ID: "replica-2", Role: "replica", Error: "timeout"},
				},
			},
			filter:   connectionFilter{instance: "replica-2"},
			wantPIDs: []int{},
			wantInstances: []live.InstanceMeta{
				{ID: "replica-2", Role: "replica", Error: "timeout"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			got := filterConnectionList(tt.input, tt.filter)
			gotPIDs := make([]int, 0, len(got.Connections))
			for _, conn := range got.Connections {
				gotPIDs = append(gotPIDs, conn.PID)
			}
			c.Assert(gotPIDs, qt.DeepEquals, tt.wantPIDs)
			if tt.wantInstances != nil {
				c.Assert(got.Instances, qt.DeepEquals, tt.wantInstances)
			}
		})
	}
}

func TestTopCmdValidation(t *testing.T) {
	tests := []struct {
		name    string
		engine  ps.DatabaseEngine
		args    []string
		wantErr string
	}{
		{
			name:    "role with instance",
			engine:  ps.DatabaseEnginePostgres,
			args:    []string{"--role", "primary", "--instance", "replica-a", "pgload", "main"},
			wantErr: "--role cannot be combined with --instance",
		},
		{
			name:    "zero interval",
			engine:  ps.DatabaseEnginePostgres,
			args:    []string{"pgload", "main", "--interval", "0s"},
			wantErr: "--interval must be greater than 0",
		},
		{
			name:    "negative interval",
			engine:  ps.DatabaseEnginePostgres,
			args:    []string{"pgload", "main", "--interval", "-1s"},
			wantErr: "--interval must be greater than 0",
		},
		{
			name:    "negative duration",
			engine:  ps.DatabaseEnginePostgres,
			args:    []string{"pgload", "main", "--duration", "-1s"},
			wantErr: "--duration must not be negative",
		},
		{
			name:    "vitess target flags on postgres",
			engine:  ps.DatabaseEnginePostgres,
			args:    []string{"pgload", "main", "--keyspace", "commerce"},
			wantErr: "--keyspace/--shard are only supported for Vitess databases",
		},
		{
			name:    "postgres filters on vitess",
			engine:  ps.DatabaseEngineMySQL,
			args:    []string{"shop", "main", "--role", "primary"},
			wantErr: "--instance/--role are only supported for Postgres databases",
		},
		{
			name:    "unknown role",
			engine:  ps.DatabaseEnginePostgres,
			args:    []string{"--role", "writer", "pgload", "main"},
			wantErr: "--role must be primary or replica",
		},
		{
			name:    "primary flag removed",
			engine:  ps.DatabaseEnginePostgres,
			args:    []string{"--primary", "pgload", "main"},
			wantErr: `unknown flag: --primary`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			cmd := topCmdForServerAndEngine("http://example.invalid", tt.engine, &bytes.Buffer{})
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			c.Assert(err, qt.ErrorMatches, tt.wantErr)
		})
	}
}

func TestTopCmdRunEHeadlessHappyPathWritesCaptureHeaderAndCapture(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, false)
	defer restoreTTY()
	server := liveConnectionsServer(t, sampleTopResponse())
	capture := filepath.Join(t.TempDir(), "trace.jsonl")
	var out bytes.Buffer
	cmd := topCmdForServer(server.URL, &out)
	cmd.SetArgs([]string{"pgload", "main", "--capture", capture, "--duration", "200ms", "--interval", "1s"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	records := readJSONLines(t, capture)
	c.Assert(records, qt.HasLen, 2)
	header := records[0]
	c.Assert(header["type"], qt.Equals, "capture_start")
	c.Assert(header["org"], qt.Equals, "acme")
	c.Assert(header["database"], qt.Equals, "pgload")
	c.Assert(header["branch"], qt.Equals, "main")
	c.Assert(header["schema_version"], qt.Equals, float64(1))
	captureRecord := capturedConnectionList(c, records[1])
	c.Assert(captureRecord["database_kind"], qt.Equals, "postgresql")
	c.Assert(captureRecord["instances"], qt.DeepEquals, []any{map[string]any{"id": "primary", "role": "primary"}})
	info, err := os.Stat(capture)
	c.Assert(err, qt.IsNil)
	c.Assert(info.Mode().Perm(), qt.Equals, os.FileMode(0o600))
}

func TestTopCmdRunERecordsRoleFilterInCaptureHeader(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, false)
	defer restoreTTY()
	server := liveConnectionsServer(t, sampleTopResponse())
	capture := filepath.Join(t.TempDir(), "trace.jsonl")
	var out bytes.Buffer
	cmd := topCmdForServer(server.URL, &out)
	cmd.SetArgs([]string{"pgload", "main", "--role", "primary", "--capture", capture, "--duration", "200ms", "--interval", "1s"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	records := readJSONLines(t, capture)
	c.Assert(records, qt.HasLen, 2)
	c.Assert(records[0]["filter"], qt.DeepEquals, map[string]any{"role": "primary"})
}

func TestTopCmdInteractiveHappyPathFetchesAndExits(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		return nil
	})
	defer restoreProgram()
	server := liveConnectionsServer(t, sampleTopResponse())
	cmd := topCmdForServer(server.URL, &bytes.Buffer{})
	cmd.SetArgs([]string{"pgload", "main"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
}

func TestTopCmdInteractivePassesTargetAndFilterToTUI(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "target",
			args: []string{"pgload", "main"},
			want: "pgload / main",
		},
		{
			name: "role filter",
			args: []string{"pgload", "main", "--role", "primary"},
			want: "filter: role=primary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			restoreTTY := setPrinterTTY(t, true)
			defer restoreTTY()
			var view string
			restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
				updated, _ := model.Update(tea.WindowSizeMsg{Width: 200, Height: 24})
				view = updated.View()
				return nil
			})
			defer restoreProgram()
			server := liveConnectionsServer(t, sampleTopResponse())
			cmd := topCmdForServer(server.URL, &bytes.Buffer{})
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			c.Assert(err, qt.IsNil)
			c.Assert(view, qt.Contains, tt.want)
		})
	}
}

func TestTopCmdKeepsPostgresConnectionCapabilities(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()
	var blockersView string
	var actionView string
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		updated := fetchInitialTopModel(t, model)
		withBlockers, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
		blockersView = withBlockers.View()
		withAction, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		actionView = withAction.View()
		return nil
	})
	defer restoreProgram()
	server := liveConnectionsServer(t, sampleTopResponseWithoutActionIDs())
	cmd := topCmdForServer(server.URL, &bytes.Buffer{})
	cmd.SetArgs([]string{"pgload", "main"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(blockersView, qt.Contains, "[blockers]")
	c.Assert(actionView, qt.Contains, "no open transaction to terminate on this connection")
}

func TestTopCmdRunsVitessTopWithTarget(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()

	server := liveConnectionsServerForTop(t, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/acme/databases/shop/branches/main/connections")
		assertTopQueryParam(c, r, "keyspace", "commerce")
		assertTopQueryParam(c, r, "shard", "-80")
		_, _ = io.WriteString(w, sampleVitessTopResponse())
	})

	var view string
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		updated := fetchInitialTopModel(t, model)
		view = updated.View()
		return nil
	})
	defer restoreProgram()

	cmd := topCmdForServerAndEngine(server.URL, ps.DatabaseEngineMySQL, &bytes.Buffer{})
	cmd.SetArgs([]string{"shop", "main", "--keyspace", "commerce", "--shard", "-80"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(view, qt.Contains, "shop / main / commerce / -80")
	c.Assert(view, qt.Contains, "PID")
	c.Assert(view, qt.Contains, "USER")
	c.Assert(view, qt.Not(qt.Contains), "BLOCK")
}

func TestTopCmdResolvesVitessTargetByPrompting(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantRequests int
		wantPrompts  []string
	}{
		{
			name:         "keyspace and shard",
			args:         []string{"shop", "main"},
			wantRequests: 3,
			wantPrompts:  []string{"Select a keyspace for connections top:", "Select a shard for connections top:"},
		},
		{
			name:         "shard only",
			args:         []string{"shop", "main", "--keyspace", "commerce"},
			wantRequests: 2,
			wantPrompts:  []string{"Select a shard for connections top:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			restoreTTY := setPrinterTTY(t, true)
			defer restoreTTY()

			var requests []string
			server := liveConnectionsServerForTop(t, func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.RawQuery)
				keyspace := r.URL.Query().Get("keyspace")
				switch {
				case keyspace == "":
					writeVitessMultipleKeyspacesResponse(w)
				case r.URL.Query().Get("shard") == "":
					c.Assert(keyspace, qt.Equals, "commerce")
					writeVitessShardedKeyspaceResponse(w)
				default:
					c.Assert(keyspace, qt.Equals, "commerce")
					c.Assert(r.URL.Query().Get("shard"), qt.Equals, "-80")
					_, _ = io.WriteString(w, sampleVitessTopResponse())
				}
			})

			var prompts []string
			restorePrompt := setSelectTargetForTest(t, func(message string, options []string) (string, error) {
				prompts = append(prompts, message)
				switch message {
				case "Select a keyspace for connections top:":
					c.Assert(options, qt.DeepEquals, []string{"commerce", "lookup"})
					return "commerce", nil
				case "Select a shard for connections top:":
					c.Assert(options, qt.DeepEquals, []string{"-80", "80-"})
					return "-80", nil
				default:
					t.Fatalf("unexpected prompt %q", message)
					return "", nil
				}
			})
			defer restorePrompt()

			restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
				return nil
			})
			defer restoreProgram()

			cmd := topCmdForServerAndEngine(server.URL, ps.DatabaseEngineMySQL, &bytes.Buffer{})
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			c.Assert(err, qt.IsNil)
			c.Assert(requests, qt.HasLen, tt.wantRequests)
			c.Assert(prompts, qt.DeepEquals, tt.wantPrompts)
		})
	}
}

func TestTopCmdNonInteractiveVitessAmbiguityDoesNotPrompt(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, false)
	defer restoreTTY()

	prompts := 0
	restorePrompt := setSelectTargetForTest(t, func(message string, options []string) (string, error) {
		prompts++
		return "", nil
	})
	defer restorePrompt()

	server := liveConnectionsServerForTop(t, func(w http.ResponseWriter, r *http.Request) {
		writeVitessMultipleKeyspacesResponse(w)
	})
	capture := filepath.Join(t.TempDir(), "trace.jsonl")
	cmd := topCmdForServerAndEngine(server.URL, ps.DatabaseEngineMySQL, &bytes.Buffer{})
	cmd.SetArgs([]string{"shop", "main", "--capture", capture, "--duration", "200ms", "--interval", "1s"})

	err := cmd.Execute()

	var httpErr *live.HTTPError
	c.Assert(errors.As(err, &httpErr), qt.IsTrue)
	c.Assert(httpErr.StatusCode, qt.Equals, http.StatusBadRequest)
	c.Assert(prompts, qt.Equals, 0)
}

func TestTopCmdHeadlessVitessMissingCaptureDoesNotPreflight(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, false)
	defer restoreTTY()

	requests := 0
	server := liveConnectionsServerForTop(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Errorf("unexpected connections request: %s", r.URL.String())
		_, _ = io.WriteString(w, sampleVitessTopResponse())
	})
	cmd := topCmdForServerAndEngine(server.URL, ps.DatabaseEngineMySQL, &bytes.Buffer{})
	cmd.SetArgs([]string{"shop", "main"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "--capture is required when running without a TTY")
	c.Assert(requests, qt.Equals, 0)
}

func TestTopCmdVitessPromptCancellation(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()

	requests := 0
	server := liveConnectionsServerForTop(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeVitessMultipleKeyspacesResponse(w)
	})

	promptErr := errors.New("prompt canceled")
	restorePrompt := setSelectTargetForTest(t, func(message string, options []string) (string, error) {
		return "", promptErr
	})
	defer restorePrompt()
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		t.Fatal("unexpected TUI launch")
		return nil
	})
	defer restoreProgram()

	cmd := topCmdForServerAndEngine(server.URL, ps.DatabaseEngineMySQL, &bytes.Buffer{})
	cmd.SetArgs([]string{"shop", "main"})

	err := cmd.Execute()

	c.Assert(errors.Is(err, promptErr), qt.IsTrue)
	c.Assert(requests, qt.Equals, 1)
}

func TestTopCmdVitessTargetRetryExhaustion(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()

	requests := 0
	server := liveConnectionsServerForTop(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeVitessShardedKeyspaceResponse(w)
	})

	prompts := 0
	restorePrompt := setSelectTargetForTest(t, func(message string, options []string) (string, error) {
		prompts++
		return "", nil
	})
	defer restorePrompt()

	cmd := topCmdForServerAndEngine(server.URL, ps.DatabaseEngineMySQL, &bytes.Buffer{})
	cmd.SetArgs([]string{"shop", "main", "--keyspace", "commerce"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "could not resolve Vitess keyspace and shard")
	c.Assert(requests, qt.Equals, 3)
	c.Assert(prompts, qt.Equals, 3)
}

func TestTopCmdVitessBackendErrorRendersInTUI(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()

	prompts := 0
	restorePrompt := setSelectTargetForTest(t, func(message string, options []string) (string, error) {
		prompts++
		return "", nil
	})
	defer restorePrompt()

	server := liveConnectionsServerForTop(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = io.WriteString(w, `{"code":"teapot","message":"not a target prompt","available":{"keyspaces":["commerce"]}}`)
	})

	var view string
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		updated := fetchInitialTopModel(t, model)
		view = updated.View()
		return nil
	})
	defer restoreProgram()

	cmd := topCmdForServerAndEngine(server.URL, ps.DatabaseEngineMySQL, &bytes.Buffer{})
	cmd.SetArgs([]string{"shop", "main"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(view, qt.Contains, "unable to load live connections")
	c.Assert(view, qt.Contains, "not a target prompt")
	c.Assert(prompts, qt.Equals, 0)
}

func TestTopCmdRunEHeadlessVitessWritesCapture(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, false)
	defer restoreTTY()

	var requests []string
	server := liveConnectionsServerForTop(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RawQuery)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/acme/databases/shop/branches/main/connections")
		assertTopQueryParam(c, r, "keyspace", "commerce")
		assertTopQueryParam(c, r, "shard", "-80")
		_, _ = io.WriteString(w, sampleVitessTopResponse())
	})
	capture := filepath.Join(t.TempDir(), "trace.jsonl")
	var out bytes.Buffer
	cmd := topCmdForServerAndEngine(server.URL, ps.DatabaseEngineMySQL, &out)
	cmd.SetArgs([]string{"shop", "main", "--keyspace", "commerce", "--shard", "-80", "--capture", capture, "--duration", "200ms", "--interval", "1s"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(requests, qt.HasLen, 2)
	records := readJSONLines(t, capture)
	c.Assert(records, qt.HasLen, 2)
	captureRecord := capturedConnectionList(c, records[1])
	c.Assert(captureRecord["database_kind"], qt.Equals, "mysql")
	c.Assert(captureRecord["topology"], qt.DeepEquals, map[string]any{
		"keyspace": "commerce",
		"shard":    "-80",
		"tablet":   "zone1-1001",
	})
}

func TestTopCmdInteractiveCaptureStartsEnabled(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()
	var view string
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		updated, _ := model.Update(tea.WindowSizeMsg{Width: 200, Height: 24})
		view = updated.View()
		return nil
	})
	defer restoreProgram()
	server := liveConnectionsServer(t, sampleTopResponse())
	capture := filepath.Join(t.TempDir(), "trace.jsonl")
	cmd := topCmdForServer(server.URL, &bytes.Buffer{})
	cmd.SetArgs([]string{"pgload", "main", "--capture", capture})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(view, qt.Contains, "rec "+capture)

	records := readJSONLines(t, capture)
	c.Assert(records, qt.HasLen, 1)
	c.Assert(records[0]["type"], qt.Equals, "capture_start")
}

func TestTopCmdInteractiveToggleCaptureCreatesDefaultFile(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()
	dir := t.TempDir()
	originalCwd, err := os.Getwd()
	c.Assert(err, qt.IsNil)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })
	c.Assert(os.Chdir(dir), qt.IsNil)

	var view string
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
		updated, _ = updated.Update(tea.WindowSizeMsg{Width: 200, Height: 24})
		view = updated.View()
		return nil
	})
	defer restoreProgram()
	server := liveConnectionsServer(t, sampleTopResponse())
	cmd := topCmdForServer(server.URL, &bytes.Buffer{})
	cmd.SetArgs([]string{"pgload", "main"})

	err = cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(view, qt.Contains, "rec connections-")
	matches, err := filepath.Glob(filepath.Join(dir, "connections-*.jsonl"))
	c.Assert(err, qt.IsNil)
	c.Assert(matches, qt.HasLen, 1)
	records := readJSONLines(t, matches[0])
	c.Assert(records, qt.HasLen, 1)
	c.Assert(records[0]["type"], qt.Equals, "capture_start")
}

func setRunTeaProgram(t *testing.T, run func(tea.Model, ...tea.ProgramOption) error) func() {
	t.Helper()
	previous := runTeaProgram
	runTeaProgram = run
	return func() { runTeaProgram = previous }
}

func liveConnectionsServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/databases/pgload/branches/main/connections" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

func liveConnectionsServerForTop(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func topCmdForServer(baseURL string, out *bytes.Buffer) *cobra.Command {
	return topCmdForServerAndEngine(baseURL, ps.DatabaseEnginePostgres, out)
}

func topCmdForServerAndEngine(baseURL string, engine ps.DatabaseEngine, out *bytes.Buffer) *cobra.Command {
	cmd := testTopCmd(&cmdutil.Helper{Config: &config.Config{
		BaseURL:        baseURL,
		Organization:   "acme",
		ServiceTokenID: "tid",
		ServiceToken:   "secret",
	}, Client: topDatabaseClient(engine)})
	cmd.SetOut(out)
	return cmd
}

func topDatabaseClient(engine ps.DatabaseEngine) func() (*ps.Client, error) {
	return func() (*ps.Client, error) {
		return &ps.Client{
			Databases: &mock.DatabaseService{
				GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
					return &ps.Database{Name: req.Database, Kind: engine}, nil
				},
			},
		}, nil
	}
}

func testTopCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := TopCmd(ch)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetErr(io.Discard)
	return cmd
}

func sampleTopResponse() string {
	return `{"type":"list","database_kind":"postgresql","captured_at":"2026-04-29T12:34:56Z","instances":[{"id":"primary","role":"primary","error":null}],"data":[{"pid":123,"instance":"primary","duration_ms":1000,"state":"active","usename":"alice","query_text":"select 1","xact_start":"2026-04-29T12:34:00.000Z","query_start":"2026-04-29T12:34:30.000Z","transaction_id":"primary-123-1777466040000000","query_id":"primary-123-1777466070000000"}]}`
}

func sampleTopResponseWithoutActionIDs() string {
	return `{"type":"list","database_kind":"postgresql","captured_at":"2026-04-29T12:34:56Z","instances":[{"id":"primary","role":"primary","error":null}],"data":[{"pid":123,"instance":"primary","duration_ms":1000,"state":"active","usename":"alice","query_text":"select 1","xact_start":"2026-04-29T12:34:00.000Z","query_start":"2026-04-29T12:34:30.000Z"}]}`
}

func sampleVitessTopResponse() string {
	return `{"type":"list","database_kind":"mysql","captured_at":"2026-06-04T12:30:00Z","instances":[],"topology":{"keyspace":"commerce","shard":"-80","tablet":"zone1-1001"},"data":[{"pid":101,"instance":"zone1-1001","duration_ms":42000,"state":"Query/executing","usename":"vt_app","datname":"checkout","client_addr":"10.0.0.1:1234","query_text":"SELECT 1","connection_id":"zone1-1001-101","query_id":"zone1-1001-101"}]}`
}

func writeVitessMultipleKeyspacesResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = io.WriteString(w, `{"code":"bad_request","message":"This database has multiple keyspaces. Specify which keyspace to target.","available":{"keyspaces":["commerce","lookup"]}}`)
}

func writeVitessShardedKeyspaceResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = io.WriteString(w, `{"code":"bad_request","message":"Keyspace 'commerce' is sharded. Specify which shard to target.","available":{"shards":["-80","80-"]}}`)
}

func fetchInitialTopModel(t *testing.T, model tea.Model) tea.Model {
	t.Helper()
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 200, Height: 24})
	cmd := updated.Init()
	if cmd == nil {
		return updated
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		if len(batch) == 0 {
			return updated
		}
		msg = batch[0]()
	}
	updated, _ = updated.Update(msg)
	return updated
}

func setSelectTargetForTest(t *testing.T, fn func(string, []string) (string, error)) func() {
	t.Helper()
	previous := selectTopTarget
	selectTopTarget = fn
	return func() { selectTopTarget = previous }
}

func assertTopQueryParam(c *qt.C, r *http.Request, key, want string) {
	c.Assert(r.URL.Query().Get(key), qt.Equals, want)
}

func capturedConnectionList(c *qt.C, record map[string]any) map[string]any {
	capture, ok := record["capture"].(map[string]any)
	c.Assert(ok, qt.IsTrue)
	return capture
}

func setPrinterTTY(t *testing.T, value bool) func() {
	t.Helper()
	previous := printer.IsTTY
	printer.IsTTY = value
	return func() { printer.IsTTY = previous }
}

func readJSONLines(t *testing.T, path string) []map[string]any {
	t.Helper()
	lines := readLines(t, path)
	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
