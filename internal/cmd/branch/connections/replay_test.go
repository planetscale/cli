package connections

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/connections/history"
	"github.com/planetscale/cli/internal/connections/tui"
)

func TestReplayClientListReturnsCapturedSnapshot(t *testing.T) {
	c := qt.New(t)
	source := newReplaySourceFromFixture(t, []live.Connection{{PID: 42, Instance: "primary"}})

	client := newReplayClient(source)
	list, err := client.List(context.Background(), live.SortByTransactionStart)

	c.Assert(err, qt.IsNil)
	c.Assert(list.Connections[0].PID, qt.Equals, 42)
}

func TestReplayClientRejectsAllActions(t *testing.T) {
	c := qt.New(t)
	source := newReplaySourceFromFixture(t, []live.Connection{{PID: 10}})
	client := newReplayClient(source)
	target := live.ActionTarget{Instance: "primary", PID: 10}

	c.Assert(client.CancelQuery(context.Background(), target), qt.ErrorMatches, "not available in replay mode")
	c.Assert(client.TerminateTransaction(context.Background(), target), qt.ErrorMatches, "not available in replay mode")
	c.Assert(client.TerminateConnection(context.Background(), target), qt.ErrorMatches, "not available in replay mode")
}

func TestTopCmdReplayRejectsMissingFile(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()

	cmd := testTopCmd(&cmdutil.Helper{Config: &config.Config{}})
	cmd.SetArgs([]string{"--replay", "/nonexistent/path/trace.jsonl"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "--replay: .*no such file.*")
}

func TestTopCmdReplayRejectsCombinationWithCapture(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()
	tmp := writeReplayFixture(t, []live.Connection{{PID: 10, Instance: "primary"}})

	cmd := testTopCmd(&cmdutil.Helper{Config: &config.Config{}})
	cmd.SetArgs([]string{"--replay", tmp, "--capture", filepath.Join(filepath.Dir(tmp), "out.jsonl")})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "--replay cannot be combined with --capture")
}

// --duration kills the TUI after wall-clock elapses regardless of paused
// state, so combining it with replay would silently dismiss the operator
// mid-step. Reject explicitly the same way --capture is rejected.
func TestTopCmdReplayRejectsCombinationWithDuration(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()
	tmp := writeReplayFixture(t, []live.Connection{{PID: 10, Instance: "primary"}})

	cmd := testTopCmd(&cmdutil.Helper{Config: &config.Config{}})
	cmd.SetArgs([]string{"--replay", tmp, "--duration", "30s"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "--duration cannot be combined with --replay")
}

func TestTopCmdReplayRejectsWithoutTTY(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, false)
	defer restoreTTY()
	tmp := writeReplayFixture(t, []live.Connection{{PID: 10, Instance: "primary"}})

	cmd := testTopCmd(&cmdutil.Helper{Config: &config.Config{}})
	cmd.SetArgs([]string{"--replay", tmp})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "--replay requires an interactive terminal")
}

func TestTopCmdReplayHappyPathRunsWithoutLiveClient(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		return nil
	})
	defer restoreProgram()
	tmp := writeReplayFixture(t, []live.Connection{{PID: 99, Instance: "primary"}})

	cmd := testTopCmd(&cmdutil.Helper{Config: &config.Config{}})
	cmd.SetArgs([]string{"--replay", tmp})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
}

func TestTopCmdReplayPreloadsAllCapturesAndStartsPaused(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()

	var capturedModel tea.Model
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		capturedModel = model
		return nil
	})
	defer restoreProgram()

	tmp := writeMultiSampleReplayFixture(t, 3)
	cmd := testTopCmd(&cmdutil.Helper{Config: &config.Config{}})
	cmd.SetArgs([]string{"--replay", tmp})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(capturedModel, qt.Not(qt.IsNil))

	m := capturedModel.(tui.Model)
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 180, Height: 24})
	c.Assert(sized.(tui.Model).View(), qt.Contains, "paused")

	// Jumping to the oldest sample proves all 3 captures were preloaded into
	// history and that the cursor isn't pinned at the latest.
	updated, _ := sized.(tui.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("{")})
	c.Assert(updated.(tui.Model).View(), qt.Contains, "step 1/3")
}

func TestTopCmdReplayUsesPostgresViewForPostgreSQLTrace(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()

	var capturedModel tea.Model
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		capturedModel = model
		return nil
	})
	defer restoreProgram()

	tmp := writeReplayFixture(t, []live.Connection{{
		PID:        99,
		Instance:   "primary",
		State:      "active",
		QueryText:  "SELECT pg_sleep(1)",
		BlockedBy:  []int{42},
		Duration:   time.Second,
		WaitEvent:  "ClientRead",
		XactStart:  ptrTime(time.Date(2026, 4, 28, 14, 59, 0, 0, time.UTC)),
		QueryStart: ptrTime(time.Date(2026, 4, 28, 14, 59, 30, 0, time.UTC)),
	}})
	cmd := testTopCmd(&cmdutil.Helper{Config: &config.Config{}})
	cmd.SetArgs([]string{"--replay", tmp})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(capturedModel, qt.Not(qt.IsNil))

	sized, _ := capturedModel.(tui.Model).Update(tea.WindowSizeMsg{Width: 160, Height: 24})
	view := sized.(tui.Model).View()

	c.Assert(view, qt.Contains, "sort xact_start")
	c.Assert(view, qt.Contains, "BLOCK")
	c.Assert(view, qt.Contains, "WAIT")
	c.Assert(view, qt.Contains, "SELECT pg_sleep(1)")
	c.Assert(view, qt.Not(qt.Contains), "TABLET")
	c.Assert(view, qt.Not(qt.Contains), "DB")
}

func TestTopCmdReplayUsesVitessViewForMySQLTrace(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()

	var capturedModel tea.Model
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		capturedModel = model
		return nil
	})
	defer restoreProgram()

	tmp := writeVitessReplayFixture(t)
	cmd := testTopCmd(&cmdutil.Helper{Config: &config.Config{}})
	cmd.SetArgs([]string{"--replay", tmp})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(capturedModel, qt.Not(qt.IsNil))

	sized, _ := capturedModel.(tui.Model).Update(tea.WindowSizeMsg{Width: 160, Height: 24})
	view := sized.(tui.Model).View()

	c.Assert(view, qt.Contains, "sorted by duration")
	c.Assert(view, qt.Contains, "TABLET")
	c.Assert(view, qt.Contains, "DB")
	c.Assert(view, qt.Contains, "SELECT 1")
	c.Assert(view, qt.Not(qt.Contains), "BLOCK")
	c.Assert(view, qt.Not(qt.Contains), "WAIT")
}

func TestReplayHeaderShowsTraceTarget(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()

	var capturedModel tea.Model
	restoreProgram := setRunTeaProgram(t, func(model tea.Model, options ...tea.ProgramOption) error {
		capturedModel = model
		return nil
	})
	defer restoreProgram()

	tmp := writeVitessReplayFixtureWithTarget(t)
	cmd := testTopCmd(&cmdutil.Helper{Config: &config.Config{}})
	cmd.SetArgs([]string{"--replay", tmp})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(capturedModel, qt.Not(qt.IsNil))

	sized, _ := capturedModel.(tui.Model).Update(tea.WindowSizeMsg{Width: 160, Height: 24})
	view := sized.(tui.Model).View()

	c.Assert(view, qt.Contains, "kind-live-connections-mysql / main / commerce / -80")
}

func writeMultiSampleReplayFixture(t *testing.T, count int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "multi.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	writer := history.NewCaptureWriter(file)
	at := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		list := live.NewConnectionList(at.Add(time.Duration(i)*time.Second), []live.Connection{{
			PID: 100 + i, Instance: "primary",
		}}, live.SortByTransactionStart)
		if err := writer.Write(history.NewCapture(list)); err != nil {
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeVitessReplayFixtureWithTarget(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vitess-target.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	writer := history.NewCaptureWriter(file)
	if err := writer.WriteCaptureStart(history.CaptureStart{
		At:       time.Date(2026, 6, 4, 12, 29, 59, 0, time.UTC),
		Database: "kind-live-connections-mysql",
		Branch:   "main",
		Target: &history.CaptureTarget{
			Keyspace: "commerce",
			Shard:    "-80",
		},
	}); err != nil {
		t.Fatal(err)
	}
	connectionID := "zone1-1001-101"
	queryID := "zone1-1001-101"
	list := live.NewConnectionList(time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC), []live.Connection{{
		PID:          101,
		Instance:     "zone1-1001",
		State:        "Query/executing",
		Duration:     42 * time.Second,
		Username:     "vt_app",
		DatabaseName: "checkout",
		QueryText:    "SELECT 1",
		ConnectionID: &connectionID,
		QueryID:      &queryID,
	}}, live.SortByDuration)
	list.DatabaseKind = live.DatabaseKindMySQL
	if err := writer.Write(history.NewCapture(list)); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeVitessReplayFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vitess.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	writer := history.NewCaptureWriter(file)
	connectionID := "zone1-1001-101"
	queryID := "zone1-1001-101"
	list := live.NewConnectionList(time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC), []live.Connection{{
		PID:          101,
		Instance:     "zone1-1001",
		State:        "Query/executing",
		Duration:     42 * time.Second,
		Username:     "vt_app",
		DatabaseName: "checkout",
		QueryText:    "SELECT 1",
		ConnectionID: &connectionID,
		QueryID:      &queryID,
	}}, live.SortByDuration)
	list.DatabaseKind = live.DatabaseKindMySQL
	list.Topology = &live.Topology{Keyspace: "commerce", Shard: "-80", Tablet: "zone1-1001"}
	if err := writer.Write(history.NewCapture(list)); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTopCmdReplayRejectsEmptyTraceFile(t *testing.T) {
	c := qt.New(t)
	restoreTTY := setPrinterTTY(t, true)
	defer restoreTTY()
	tmp := filepath.Join(t.TempDir(), "empty.jsonl")
	if err := os.WriteFile(tmp, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := testTopCmd(&cmdutil.Helper{Config: &config.Config{}})
	cmd.SetArgs([]string{"--replay", tmp})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "--replay: capture file contains no replayable snapshots")
}

func writeReplayFixture(t *testing.T, connections []live.Connection) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "trace.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	writer := history.NewCaptureWriter(file)
	list := live.NewConnectionList(time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC), connections, live.SortByTransactionStart)
	list.DatabaseKind = live.DatabaseKindPostgreSQL
	if err := writer.Write(history.NewCapture(list)); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func newReplaySourceFromFixture(t *testing.T, connections []live.Connection) *history.ReplaySource {
	t.Helper()
	var buffer bytes.Buffer
	writer := history.NewCaptureWriter(&buffer)
	list := live.NewConnectionList(time.Date(2026, 4, 28, 15, 0, 0, 0, time.UTC), connections, live.SortByTransactionStart)
	if err := writer.Write(history.NewCapture(list)); err != nil {
		t.Fatal(err)
	}
	source, err := history.NewReplaySource(&buffer)
	if err != nil {
		t.Fatal(err)
	}
	return source
}
