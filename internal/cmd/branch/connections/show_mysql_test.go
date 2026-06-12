package connections

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/printer"
)

func TestPrintListJSONIncludesDatabaseAndTopology(t *testing.T) {
	c := qt.New(t)

	got := printListForTest(c, printer.JSON, mysqlConnectionList(), commerceTopology())

	var payload struct {
		DatabaseKind live.DatabaseKind `json:"database_kind"`
		Topology     *ListTopology     `json:"topology"`
		Connections  []struct {
			DatabaseName    string `json:"database"`
			ApplicationName string `json:"application_name"`
		} `json:"connections"`
	}
	decodeJSONForTest(c, got, &payload)

	c.Assert(payload.DatabaseKind, qt.Equals, live.DatabaseKindMySQL)
	c.Assert(payload.Topology, qt.DeepEquals, &ListTopology{Keyspace: "commerce", Shard: "-80", Tablet: "zone1-1001"})
	c.Assert(payload.Connections, qt.HasLen, 1)
	c.Assert(payload.Connections[0].DatabaseName, qt.Equals, "checkout")
	c.Assert(payload.Connections[0].ApplicationName, qt.Equals, "")
}

func TestPrintListHumanIncludesDatabaseAndTopology(t *testing.T) {
	c := qt.New(t)

	got := printListForTest(c, printer.Human, mysqlConnectionList(), commerceTopology())

	c.Assert(got, qt.Contains, "topology:\n")
	c.Assert(got, qt.Contains, "  keyspace: commerce\n")
	c.Assert(got, qt.Contains, "  shard: -80\n")
	c.Assert(got, qt.Contains, "  tablet: zone1-1001\n")
	c.Assert(got, qt.Contains, "database:        checkout\n")
}

func TestPrintListVitessHumanUsesTabletLabels(t *testing.T) {
	c := qt.New(t)

	got := printListForTest(c, printer.Human, mysqlConnectionList(), commerceTopology())

	c.Assert(got, qt.Contains, "tablet:          zone1-1001\n")
	c.Assert(got, qt.Contains, "database:        checkout\n")
	c.Assert(got, qt.Not(qt.Contains), "instance:        zone1-1001\n")
	c.Assert(got, qt.Not(qt.Contains), "role:")
	c.Assert(got, qt.Not(qt.Contains), "Database:        checkout\n")
}

func TestPrintListCSVIncludesDatabaseAndTopology(t *testing.T) {
	c := qt.New(t)

	got := printListForTest(c, printer.CSV, mysqlConnectionList(), commerceTopology())
	rows := readCSVForTest(c, got)

	headers := rows[0]
	c.Assert(headers, qt.Contains, "keyspace")
	c.Assert(headers, qt.Contains, "shard")
	c.Assert(headers, qt.Contains, "tablet")
	c.Assert(headers, qt.Contains, "database")
	c.Assert(rows[1], qt.Contains, "commerce")
	c.Assert(rows[1], qt.Contains, "-80")
	c.Assert(rows[1], qt.Contains, "zone1-1001")
	c.Assert(rows[1], qt.Contains, "checkout")
}

func TestPrintListCSVUsesConsistentHeaders(t *testing.T) {
	c := qt.New(t)

	got := printListForTest(c, printer.CSV, mysqlConnectionList(), commerceTopology())
	rows := readCSVForTest(c, got)

	c.Assert(rows[0], qt.DeepEquals, []string{
		"keyspace",
		"shard",
		"tablet",
		"pid",
		"instance",
		"role",
		"state",
		"duration_ms",
		"wait_event_type",
		"wait_event",
		"username",
		"application_name",
		"database",
		"client_addr",
		"query_text",
		"blocked_by",
		"query_id",
		"transaction_id",
		"connection_id",
	})
}

func TestPrintListMySQLJSONIncludesDatabaseWithoutTopology(t *testing.T) {
	c := qt.New(t)

	got := printListForTest(c, printer.JSON, mysqlConnectionList(), ListTopology{})

	var payload struct {
		DatabaseKind live.DatabaseKind `json:"database_kind"`
		Topology     *ListTopology     `json:"topology"`
		Connections  []struct {
			DatabaseName string `json:"database"`
		} `json:"connections"`
	}
	decodeJSONForTest(c, got, &payload)

	c.Assert(payload.DatabaseKind, qt.Equals, live.DatabaseKindMySQL)
	c.Assert(payload.Topology, qt.IsNil)
	c.Assert(payload.Connections, qt.HasLen, 1)
	c.Assert(payload.Connections[0].DatabaseName, qt.Equals, "checkout")
}

func TestPrintListMySQLHumanIncludesDatabaseWithoutTopology(t *testing.T) {
	c := qt.New(t)

	got := printListForTest(c, printer.Human, mysqlConnectionList(), ListTopology{})

	c.Assert(got, qt.Contains, "database:        checkout\n")
	c.Assert(got, qt.Not(qt.Contains), "topology:\n")
}

func TestPrintListMySQLCSVIncludesDatabaseWithoutTopology(t *testing.T) {
	c := qt.New(t)

	got := printListForTest(c, printer.CSV, mysqlConnectionList(), ListTopology{})
	rows := readCSVForTest(c, got)

	headers := rows[0]
	c.Assert(headers, qt.Contains, "database")
	c.Assert(headers, qt.Not(qt.Contains), "keyspace")
	c.Assert(headers, qt.Not(qt.Contains), "shard")
	c.Assert(headers, qt.Not(qt.Contains), "tablet")
	c.Assert(rows[1], qt.Contains, "checkout")
}

func TestPrintListUsesTopologyFromList(t *testing.T) {
	c := qt.New(t)
	list := mysqlConnectionList(func(list *live.ConnectionList) {
		list.Topology = &live.Topology{
			Keyspace: "commerce",
			Shard:    "-80",
			Tablet:   "zone1-1001",
		}
	})

	got := printListForTest(c, printer.JSON, list, ListTopology{})

	var payload struct {
		Topology *ListTopology `json:"topology"`
	}
	decodeJSONForTest(c, got, &payload)
	c.Assert(payload.Topology, qt.DeepEquals, &ListTopology{Keyspace: "commerce", Shard: "-80", Tablet: "zone1-1001"})
}

func TestPrintListExplicitTopologyOverridesListTopology(t *testing.T) {
	c := qt.New(t)
	list := mysqlConnectionList(func(list *live.ConnectionList) {
		list.Topology = &live.Topology{
			Keyspace: "stale",
			Shard:    "0",
			Tablet:   "stale-tablet",
		}
	})

	got := printListForTest(c, printer.JSON, list, commerceTopology())

	var payload struct {
		Topology *ListTopology `json:"topology"`
	}
	decodeJSONForTest(c, got, &payload)
	c.Assert(payload.Topology, qt.DeepEquals, &ListTopology{Keyspace: "commerce", Shard: "-80", Tablet: "zone1-1001"})
}

func printListForTest(c *qt.C, format printer.Format, list live.ConnectionList, topology ListTopology) string {
	var out bytes.Buffer
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	ch := &cmdutil.Helper{Printer: p}

	c.Assert(PrintList(ch, list, topology), qt.IsNil)
	return out.String()
}

func mysqlConnectionList(overrides ...func(*live.ConnectionList)) live.ConnectionList {
	connectionID := "101"
	list := live.ConnectionList{
		CapturedAt:   time.Date(2026, 4, 29, 12, 34, 56, 0, time.UTC),
		DatabaseKind: live.DatabaseKindMySQL,
		Connections: []live.Connection{
			{
				PID:          101,
				Instance:     "zone1-1001",
				Username:     "vt_app",
				DatabaseName: "checkout",
				ClientAddr:   "10.0.0.12:54231",
				State:        "Query",
				Duration:     42 * time.Second,
				ConnectionID: &connectionID,
				QueryText:    "select * from orders",
			},
		},
	}
	for _, override := range overrides {
		override(&list)
	}
	return list
}

func commerceTopology() ListTopology {
	return ListTopology{Keyspace: "commerce", Shard: "-80", Tablet: "zone1-1001"}
}

func decodeJSONForTest(c *qt.C, raw string, out any) {
	c.Assert(json.Unmarshal([]byte(raw), out), qt.IsNil)
}

func readCSVForTest(c *qt.C, raw string) [][]string {
	rows, err := csv.NewReader(strings.NewReader(raw)).ReadAll()
	c.Assert(err, qt.IsNil)
	return rows
}

func TestVitessShowSortsByDurationDesc(t *testing.T) {
	c := qt.New(t)
	list := live.ConnectionList{
		DatabaseKind: live.DatabaseKindMySQL,
		Connections: []live.Connection{
			{PID: 1, Duration: 5 * time.Second},
			{PID: 2, Duration: 50 * time.Second},
			{PID: 3, Duration: 20 * time.Second},
		},
	}
	sortListForDisplay(&list)
	c.Assert([]int{list.Connections[0].PID, list.Connections[1].PID, list.Connections[2].PID}, qt.DeepEquals, []int{2, 3, 1})

	pg := live.ConnectionList{
		DatabaseKind: live.DatabaseKindPostgreSQL,
		Connections: []live.Connection{
			{PID: 1, Duration: 5 * time.Second},
			{PID: 2, Duration: 50 * time.Second},
		},
	}
	sortListForDisplay(&pg)
	c.Assert(pg.Connections[0].PID, qt.Equals, 1)
}
