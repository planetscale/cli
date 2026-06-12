package connections

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func TestToPrintableListMapsEnvelopeAndUsesConnectionsForCSV(t *testing.T) {
	c := qt.New(t)
	queryID := "replica-321-query"
	transactionID := "replica-321-transaction"
	connectionID := "replica-321-connection"
	capturedAt := time.Date(2026, 4, 29, 12, 34, 56, 0, time.UTC)

	got := toPrintableList(live.ConnectionList{
		CapturedAt: capturedAt,
		Instances: []live.InstanceMeta{
			{ID: "primary", Role: "primary"},
			{ID: "replica-a", Role: "replica", Error: "timeout"},
		},
		Connections: []live.Connection{
			{
				PID:             321,
				Instance:        "replica-a",
				InstanceRole:    "replica",
				State:           "idle",
				Duration:        3*time.Second + 250*time.Millisecond,
				WaitEventType:   "Client",
				WaitEvent:       "ClientRead",
				Username:        "brett",
				ApplicationName: "psql",
				ClientAddr:      "192.0.2.15",
				QueryText:       "select now()",
				BlockedBy:       []int{456, 789},
				QueryID:         &queryID,
				TransactionID:   &transactionID,
				ConnectionID:    &connectionID,
			},
		},
	}, ListTopology{})

	c.Assert(got.CapturedAt, qt.Equals, capturedAt)
	c.Assert(got.Instances, qt.DeepEquals, []printableInstance{
		{ID: "primary", Role: "primary"},
		{ID: "replica-a", Role: "replica", Error: "timeout"},
	})
	c.Assert(got.Connections, qt.DeepEquals, []printableConnection{
		{
			PID:             321,
			Instance:        "replica-a",
			InstanceRole:    "replica",
			State:           "idle",
			DurationMS:      3250,
			WaitEventType:   "Client",
			WaitEvent:       "ClientRead",
			Username:        "brett",
			ApplicationName: "psql",
			ClientAddr:      "192.0.2.15",
			QueryText:       "select now()",
			BlockedBy:       []int{456, 789},
			QueryID:         &queryID,
			TransactionID:   &transactionID,
			ConnectionID:    &connectionID,
		},
	})
	c.Assert(got.MarshalCSVValue(), qt.DeepEquals, got.Connections)
}

func TestPrintHumanConnectionListUsesVerticalRecordsWithoutTruncatingQueryText(t *testing.T) {
	c := qt.New(t)
	queryID := "primary-123-query"
	transactionID := "primary-123-transaction"
	connectionID := "primary-123-connection"
	longQuery := "select " + string(bytes.Repeat([]byte("really_long_expression + "), 20)) + "42"
	var out bytes.Buffer

	printHumanConnectionList(&out, live.ConnectionList{
		CapturedAt: time.Date(2026, 4, 29, 12, 34, 56, 0, time.UTC),
		Connections: []live.Connection{
			{
				PID:             123,
				Instance:        "primary",
				InstanceRole:    "primary",
				State:           "active",
				Duration:        2*time.Second + 500*time.Millisecond,
				WaitEventType:   "Lock",
				WaitEvent:       "transactionid",
				Username:        "brett",
				ApplicationName: "psql",
				ClientAddr:      "127.0.0.1",
				QueryText:       longQuery,
				BlockedBy:       []int{456, 789},
				QueryID:         &queryID,
				TransactionID:   &transactionID,
				ConnectionID:    &connectionID,
			},
		},
	}, ListTopology{})

	got := out.String()
	c.Assert(got, qt.Contains, "*************************** 1. row ***************************\n")
	c.Assert(got, qt.Contains, "pid:             123\n")
	c.Assert(got, qt.Contains, "blocked_by:      456,789\n")
	c.Assert(got, qt.Contains, "query_id:        primary-123-query\n")
	c.Assert(got, qt.Contains, "transaction_id:  primary-123-transaction\n")
	c.Assert(got, qt.Contains, "connection_id:   primary-123-connection\n")
	c.Assert(got, qt.Contains, "query:\n"+longQuery+"\n")
}

func TestPrintListPostgresHumanUnchanged(t *testing.T) {
	c := qt.New(t)
	queryID := "primary-123-query"
	transactionID := "primary-123-transaction"
	connectionID := "primary-123-connection"

	got := printListForTest(c, printer.Human, live.ConnectionList{
		CapturedAt: time.Date(2026, 4, 29, 12, 34, 56, 0, time.UTC),
		Connections: []live.Connection{
			{
				PID:             123,
				Instance:        "primary",
				InstanceRole:    "primary",
				State:           "active",
				Duration:        2 * time.Second,
				Username:        "brett",
				ApplicationName: "psql",
				QueryText:       "select 1",
				QueryID:         &queryID,
				TransactionID:   &transactionID,
				ConnectionID:    &connectionID,
			},
		},
	}, ListTopology{})

	c.Assert(got, qt.Contains, "instance:        primary\n")
	c.Assert(got, qt.Contains, "role:            primary\n")
	c.Assert(got, qt.Not(qt.Contains), "tablet:          primary\n")
	c.Assert(got, qt.Not(qt.Contains), "Database:")
}

func TestPrintHumanConnectionListEmpty(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer

	printHumanConnectionList(&out, live.ConnectionList{
		CapturedAt: time.Date(2026, 4, 29, 12, 34, 56, 0, time.UTC),
	}, ListTopology{})

	got := out.String()
	c.Assert(got, qt.Contains, "captured_at: 2026-04-29T12:34:56Z\n")
	c.Assert(got, qt.Contains, "No live connections found.\n")
}

func TestPrintHumanConnectionListFlagsUnreachableInstances(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer

	printHumanConnectionList(&out, live.ConnectionList{
		CapturedAt: time.Date(2026, 4, 29, 12, 34, 56, 0, time.UTC),
		Instances: []live.InstanceMeta{
			{ID: "primary", Role: "primary"},
			{ID: "replica-b", Role: "replica", Error: "timeout"},
		},
	}, ListTopology{})

	c.Assert(out.String(), qt.Contains, "warning: partial results, unreachable instances: replica-b\n")
}

func TestShowCmdHumanOutputFetchesOnceAndPrintsVerticalRecords(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	var seenPath string
	server := liveConnectionsListServer(t, sampleListCmdResponse(), &seenPath)
	cmd := showCmdForServer(server.URL, printer.Human, &out)

	cmd.SetArgs([]string{"pgload", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(seenPath, qt.Equals, "/v1/organizations/acme/databases/pgload/branches/main/connections")
	got := out.String()
	c.Assert(got, qt.Contains, "*************************** 1. row ***************************\n")
	c.Assert(got, qt.Contains, "pid:             123\n")
	c.Assert(got, qt.Contains, "query_id:        primary-123-q\n")
	c.Assert(got, qt.Contains, "transaction_id:  primary-123-t\n")
	c.Assert(got, qt.Contains, "connection_id:   primary-123-c\n")
	c.Assert(got, qt.Not(qt.Contains), "database:")
	c.Assert(got, qt.Contains, "query:\nSELECT pg_sleep(600)\n")
}

func TestShowCmdCSVOutput(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	server := liveConnectionsListServer(t, sampleListCmdResponse(), nil)
	cmd := showCmdForServer(server.URL, printer.CSV, &out)

	cmd.SetArgs([]string{"pgload", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	got := out.String()
	c.Assert(got, qt.Not(qt.Contains), "database")
	c.Assert(got, qt.Not(qt.Contains), "Database")
	c.Assert(got, qt.Contains, "123,primary,primary,active,664000")
}

func TestShowCmdJSONOutputPrintsEnvelope(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	server := liveConnectionsListServer(t, sampleListCmdResponse(), nil)
	cmd := showCmdForServer(server.URL, printer.JSON, &out)

	cmd.SetArgs([]string{"pgload", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	var got printableList
	c.Assert(json.Unmarshal(out.Bytes(), &got), qt.IsNil)
	c.Assert(got.CapturedAt.IsZero(), qt.Equals, false)
	c.Assert(got.Instances, qt.HasLen, 1)
	c.Assert(got.Connections, qt.HasLen, 1)
	c.Assert(got.Connections[0].PID, qt.Equals, 123)
	c.Assert(got.Connections[0].DurationMS, qt.Equals, int64(664000))
	c.Assert(got.Connections[0].BlockedBy, qt.DeepEquals, []int{})
	c.Assert(got.Connections[0].QueryID, qt.DeepEquals, stringPtr("primary-123-q"))
	c.Assert(got.Connections[0].TransactionID, qt.DeepEquals, stringPtr("primary-123-t"))
	c.Assert(got.Connections[0].ConnectionID, qt.DeepEquals, stringPtr("primary-123-c"))
	c.Assert(out.String(), qt.Contains, `"database_kind": "postgresql"`)
}

func TestShowCmdFiltersByRoleAndInstance(t *testing.T) {
	c := qt.New(t)
	server := liveConnectionsListServer(t, sampleFilteredListCmdResponse(), nil)

	var primaryOut bytes.Buffer
	primaryCmd := showCmdForServer(server.URL, printer.JSON, &primaryOut)
	primaryCmd.SetArgs([]string{"--role", "primary", "pgload", "main"})
	c.Assert(primaryCmd.Execute(), qt.IsNil)
	var primary printableList
	c.Assert(json.Unmarshal(primaryOut.Bytes(), &primary), qt.IsNil)
	c.Assert(primary.Connections, qt.HasLen, 1)
	c.Assert(primary.Connections[0].Instance, qt.Equals, "primary")

	var replicaOut bytes.Buffer
	replicaCmd := showCmdForServer(server.URL, printer.JSON, &replicaOut)
	replicaCmd.SetArgs([]string{"--role", "replica", "pgload", "main"})
	c.Assert(replicaCmd.Execute(), qt.IsNil)
	var replica printableList
	c.Assert(json.Unmarshal(replicaOut.Bytes(), &replica), qt.IsNil)
	c.Assert(replica.Connections, qt.HasLen, 1)
	c.Assert(replica.Connections[0].Instance, qt.Equals, "replica-a")

	var instanceOut bytes.Buffer
	instanceCmd := showCmdForServer(server.URL, printer.JSON, &instanceOut)
	instanceCmd.SetArgs([]string{"--instance", "replica-a", "pgload", "main"})
	c.Assert(instanceCmd.Execute(), qt.IsNil)
	var instance printableList
	c.Assert(json.Unmarshal(instanceOut.Bytes(), &instance), qt.IsNil)
	c.Assert(instance.Connections, qt.HasLen, 1)
	c.Assert(instance.Connections[0].Instance, qt.Equals, "replica-a")
}

func TestShowUnknownInstanceReturnsError(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	server := liveConnectionsListServer(t, sampleFilteredListCmdResponse(), nil)
	cmd := showCmdForServer(server.URL, printer.JSON, &out)
	cmd.SetArgs([]string{"--instance", "missing", "pgload", "main"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `unknown instance "missing" \(valid instances: primary, replica-a\)`)
	c.Assert(out.String(), qt.Equals, "")
}

func TestShowValidInstanceWithNoConnectionsSucceeds(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	server := liveConnectionsListServer(t, samplePrimaryOnlyListCmdResponse(), nil)
	cmd := showCmdForServer(server.URL, printer.JSON, &out)
	cmd.SetArgs([]string{"--instance", "replica-a", "pgload", "main"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	var got printableList
	c.Assert(json.Unmarshal(out.Bytes(), &got), qt.IsNil)
	c.Assert(got.Instances, qt.DeepEquals, []printableInstance{{ID: "replica-a", Role: "replica"}})
	c.Assert(got.Connections, qt.HasLen, 0)
}

func TestShowForbiddenReturnsNonLeakyPermissionError(t *testing.T) {
	c := qt.New(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/acme/databases/pgload/branches/main/connections")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"message":"denied by infra policy on tablet zone1-1001"}`)
	}))
	t.Cleanup(server.Close)

	var out bytes.Buffer
	cmd := showCmdForServer(server.URL, printer.JSON, &out)
	cmd.SetArgs([]string{"pgload", "main"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "permission denied: you don't have permission to view live connections")
	c.Assert(err.Error(), qt.Not(qt.Contains), "zone1-1001")
	c.Assert(out.String(), qt.Equals, "")
}

func TestShowCmdRejectsRoleWithInstance(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	cmd := showCmdForServer("http://127.0.0.1:1", printer.JSON, &out)
	cmd.SetArgs([]string{"--role", "primary", "--instance", "replica-a", "pgload", "main"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "--role cannot be combined with --instance")
}

func TestShowCmdRejectsUnknownRole(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	cmd := showCmdForServer("http://127.0.0.1:1", printer.JSON, &out)
	cmd.SetArgs([]string{"--role", "writer", "pgload", "main"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "--role must be primary or replica")
}

func TestShowCmdRejectsPrimaryFlag(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	cmd := showCmdForServer("http://127.0.0.1:1", printer.JSON, &out)
	cmd.SetArgs([]string{"--primary", "pgload", "main"})

	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `unknown flag: --primary`)
}

func liveConnectionsListServer(t *testing.T, body string, seenPath *string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/acme/databases/pgload/branches/main/connections" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if seenPath != nil {
			*seenPath = r.URL.Path
		}
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

// showCmdForServer builds a minimal cobra command that exercises the Postgres
// RunList path (filtering, printing, error sanitization) against a test server.
// The shipping command is branch.ConnectionsShowCmd, which lives one package up
// and adds engine detection; these tests cover the engine-agnostic list path it
// delegates to.
func showCmdForServer(baseURL string, format printer.Format, out *bytes.Buffer) *cobra.Command {
	ch := liveConnectionsTestHelper(baseURL, format, out)
	var flags struct {
		instance string
		role     string
	}
	cmd := &cobra.Command{
		Use:  "show <database> <branch>",
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunList(cmd.Context(), ch, args[0], args[1], ConnectionFilter{Instance: flags.instance, Role: flags.role}, ConnectionTarget{})
		},
	}
	cmd.Flags().StringVar(&flags.instance, "instance", "", "Filter the list to a single instance.")
	cmd.Flags().StringVar(&flags.role, "role", "", "Filter the list to rows whose instance role is primary or replica.")
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetErr(io.Discard)
	return cmd
}

func liveConnectionsTestHelper(baseURL string, format printer.Format, out *bytes.Buffer) *cmdutil.Helper {
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

func sampleListCmdResponse() string {
	return `{"type":"list","database_kind":"postgresql","captured_at":"2026-04-29T12:34:56Z","instances":[{"id":"primary","role":"primary","error":null}],"data":[{"pid":123,"instance":"primary","duration_ms":664000,"state":"active","usename":"alice","application_name":"psql","client_addr":"10.0.0.1","wait_event_type":"Client","wait_event":"ClientRead","query_text":"SELECT pg_sleep(600)","xact_start":"2026-04-29T12:23:52Z","query_start":"2026-04-29T12:23:52Z","query_id":"primary-123-q","transaction_id":"primary-123-t","connection_id":"primary-123-c"}]}`
}

func sampleFilteredListCmdResponse() string {
	return `{
		"type": "list",
		"captured_at": "2026-04-29T12:34:56Z",
		"instances": [
			{"id": "primary", "role": "primary", "error": null},
			{"id": "replica-a", "role": "replica", "error": null}
		],
		"data": [
			{
				"pid": 123,
				"instance": "primary",
				"duration_ms": 664000,
				"state": "active",
				"usename": "alice",
				"application_name": "psql",
				"client_addr": "10.0.0.1",
				"query_text": "SELECT pg_sleep(600)",
				"xact_start": "2026-04-29T12:23:52Z",
				"query_start": "2026-04-29T12:23:52Z",
				"query_id": "primary-123-q",
				"transaction_id": "primary-123-t",
				"connection_id": "primary-123-c"
			},
			{
				"pid": 456,
				"instance": "replica-a",
				"duration_ms": 2000,
				"state": "idle",
				"usename": "bob",
				"application_name": "psql",
				"client_addr": "10.0.0.2",
				"query_text": "SELECT 1",
				"xact_start": "2026-04-29T12:34:54Z",
				"query_start": "2026-04-29T12:34:54Z",
				"query_id": "replica-456-q",
				"transaction_id": "replica-456-t",
				"connection_id": "replica-456-c"
			}
		]
	}`
}

func samplePrimaryOnlyListCmdResponse() string {
	return `{
		"type": "list",
		"captured_at": "2026-04-29T12:34:56Z",
		"instances": [
			{"id": "primary", "role": "primary", "error": null},
			{"id": "replica-a", "role": "replica", "error": null}
		],
		"data": [
			{
				"pid": 123,
				"instance": "primary",
				"duration_ms": 664000,
				"state": "active",
				"usename": "alice",
				"application_name": "psql",
				"client_addr": "10.0.0.1",
				"query_text": "SELECT pg_sleep(600)",
				"xact_start": "2026-04-29T12:23:52Z",
				"query_start": "2026-04-29T12:23:52Z",
				"query_id": "primary-123-q",
				"transaction_id": "primary-123-t",
				"connection_id": "primary-123-c"
			}
		]
	}`
}

func stringPtr(value string) *string {
	return &value
}
