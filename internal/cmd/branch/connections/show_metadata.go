package connections

import live "github.com/planetscale/cli/internal/connections"

// ListTopology describes the Vitess tablet selected for a live connection list.
type ListTopology struct {
	Keyspace string `json:"keyspace,omitempty"`
	Shard    string `json:"shard,omitempty"`
	Tablet   string `json:"tablet,omitempty"`
}

func (t ListTopology) isEmpty() bool {
	return t.Keyspace == "" && t.Shard == "" && t.Tablet == ""
}

func resolveListTopology(list live.ConnectionList, topology ListTopology) ListTopology {
	if !topology.isEmpty() || list.Topology == nil {
		return topology
	}
	return ListTopology{
		Keyspace: list.Topology.Keyspace,
		Shard:    list.Topology.Shard,
		Tablet:   list.Topology.Tablet,
	}
}

type printableCSVConnectionWithDatabase struct {
	PID             int     `csv:"pid" header:"pid" json:"pid"`
	Instance        string  `csv:"instance" header:"instance" json:"instance"`
	InstanceRole    string  `csv:"role" header:"role" json:"instance_role"`
	State           string  `csv:"state" header:"state" json:"state"`
	DurationMS      int64   `csv:"duration_ms" header:"duration_ms" json:"duration_ms"`
	WaitEventType   string  `csv:"wait_event_type" header:"wait_event_type" json:"wait_event_type"`
	WaitEvent       string  `csv:"wait_event" header:"wait_event" json:"wait_event"`
	Username        string  `csv:"username" header:"username" json:"username"`
	ApplicationName string  `csv:"application_name" header:"application_name" json:"application_name"`
	DatabaseName    string  `csv:"database" header:"database" json:"database,omitempty"`
	ClientAddr      string  `csv:"client_addr" header:"client_addr" json:"client_addr"`
	QueryText       string  `csv:"query_text" header:"query_text" json:"query_text"`
	BlockedBy       []int   `csv:"blocked_by" header:"blocked_by" json:"blocked_by"`
	QueryID         *string `csv:"query_id" header:"query_id" json:"query_id"`
	TransactionID   *string `csv:"transaction_id" header:"transaction_id" json:"transaction_id"`
	ConnectionID    *string `csv:"connection_id" header:"connection_id" json:"connection_id"`
}

type printableConnectionWithTopology struct {
	Keyspace        string  `csv:"keyspace" header:"keyspace" json:"keyspace"`
	Shard           string  `csv:"shard" header:"shard" json:"shard"`
	Tablet          string  `csv:"tablet" header:"tablet" json:"tablet"`
	PID             int     `csv:"pid" header:"pid" json:"pid"`
	Instance        string  `csv:"instance" header:"instance" json:"instance"`
	InstanceRole    string  `csv:"role" header:"role" json:"instance_role"`
	State           string  `csv:"state" header:"state" json:"state"`
	DurationMS      int64   `csv:"duration_ms" header:"duration_ms" json:"duration_ms"`
	WaitEventType   string  `csv:"wait_event_type" header:"wait_event_type" json:"wait_event_type"`
	WaitEvent       string  `csv:"wait_event" header:"wait_event" json:"wait_event"`
	Username        string  `csv:"username" header:"username" json:"username"`
	ApplicationName string  `csv:"application_name" header:"application_name" json:"application_name"`
	DatabaseName    string  `csv:"database" header:"database" json:"database,omitempty"`
	ClientAddr      string  `csv:"client_addr" header:"client_addr" json:"client_addr"`
	QueryText       string  `csv:"query_text" header:"query_text" json:"query_text"`
	BlockedBy       []int   `csv:"blocked_by" header:"blocked_by" json:"blocked_by"`
	QueryID         *string `csv:"query_id" header:"query_id" json:"query_id"`
	TransactionID   *string `csv:"transaction_id" header:"transaction_id" json:"transaction_id"`
	ConnectionID    *string `csv:"connection_id" header:"connection_id" json:"connection_id"`
}

func (p printableList) connectionsWithDatabase() []printableCSVConnectionWithDatabase {
	connections := make([]printableCSVConnectionWithDatabase, 0, len(p.Connections))
	for _, conn := range p.Connections {
		connections = append(connections, toPrintableCSVConnectionWithDatabase(conn))
	}
	return connections
}

func (p printableList) connectionsWithTopology() []printableConnectionWithTopology {
	connections := make([]printableConnectionWithTopology, 0, len(p.Connections))
	for _, conn := range p.Connections {
		var keyspace, shard, tablet string
		if p.Topology != nil {
			keyspace = p.Topology.Keyspace
			shard = p.Topology.Shard
			tablet = p.Topology.Tablet
		}
		connections = append(connections, printableConnectionWithTopology{
			Keyspace:        keyspace,
			Shard:           shard,
			Tablet:          tablet,
			PID:             conn.PID,
			Instance:        conn.Instance,
			InstanceRole:    conn.InstanceRole,
			State:           conn.State,
			DurationMS:      conn.DurationMS,
			WaitEventType:   conn.WaitEventType,
			WaitEvent:       conn.WaitEvent,
			Username:        conn.Username,
			ApplicationName: conn.ApplicationName,
			DatabaseName:    conn.DatabaseName,
			ClientAddr:      conn.ClientAddr,
			QueryText:       conn.QueryText,
			BlockedBy:       conn.BlockedBy,
			QueryID:         conn.QueryID,
			TransactionID:   conn.TransactionID,
			ConnectionID:    conn.ConnectionID,
		})
	}
	return connections
}

func toPrintableCSVConnectionWithDatabase(conn printableConnection) printableCSVConnectionWithDatabase {
	// Identical fields; the conversion re-applies the target type's csv tags so
	// DatabaseName is emitted in CSV output (the base type hides it with csv:"-").
	return printableCSVConnectionWithDatabase(conn)
}
