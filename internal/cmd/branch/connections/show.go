package connections

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/printer"
)

// ConnectionFilter filters live connection rows by instance or instance role.
type ConnectionFilter struct {
	Instance string
	Role     string
}

// ConnectionTarget identifies an optional Vitess target for connection commands.
type ConnectionTarget struct {
	Keyspace string
	Shard    string
}

func (f ConnectionFilter) connectionFilter() connectionFilter {
	return connectionFilter{instance: f.Instance, role: f.Role}
}

// ValidateConnectionFilter validates mutually exclusive live connection filters.
func ValidateConnectionFilter(filter ConnectionFilter) error {
	return validateConnectionFilter(filter.Instance, filter.Role)
}

// RunList fetches and prints one live connection list.
func RunList(ctx context.Context, ch *cmdutil.Helper, database, branch string, filter ConnectionFilter, target ConnectionTarget) error {
	if err := ValidateConnectionFilter(filter); err != nil {
		return err
	}
	return runList(ctx, ch, database, branch, filter.connectionFilter(), target)
}

func runList(ctx context.Context, ch *cmdutil.Helper, database, branch string, filter connectionFilter, target ConnectionTarget) error {
	client, err := newConnectionsClient(ch, database, branch, target)
	if err != nil {
		return err
	}

	list, err := filteredLister{client: client, filter: filter}.List(ctx, live.SortByTransactionStart)
	if err != nil {
		return live.UserFacingError(err, "view")
	}
	sortListForDisplay(&list)

	return PrintList(ch, list, ListTopology{})
}

// PrintList prints a live connection list in the configured CLI format.
func PrintList(ch *cmdutil.Helper, list live.ConnectionList, topology ListTopology) error {
	topology = resolveListTopology(list, topology)

	if ch.Printer.Format() == printer.Human {
		var out strings.Builder
		printHumanConnectionList(&out, list, topology)
		ch.Printer.Print(out.String())
		return nil
	}

	return ch.Printer.PrintResource(toPrintableList(list, topology))
}

type printableList struct {
	DatabaseKind live.DatabaseKind     `json:"database_kind,omitempty"`
	CapturedAt   time.Time             `json:"captured_at"`
	Topology     *ListTopology         `json:"topology,omitempty"`
	Instances    []printableInstance   `json:"instances"`
	Connections  []printableConnection `json:"connections"`
}

type printableInstance struct {
	ID    string `json:"id"`
	Role  string `json:"role"`
	Error string `json:"error,omitempty"`
}

type printableConnection struct {
	PID             int     `csv:"pid" header:"pid" json:"pid"`
	Instance        string  `csv:"instance" header:"instance" json:"instance"`
	InstanceRole    string  `csv:"role" header:"role" json:"instance_role"`
	State           string  `csv:"state" header:"state" json:"state"`
	DurationMS      int64   `csv:"duration_ms" header:"duration_ms" json:"duration_ms"`
	WaitEventType   string  `csv:"wait_event_type" header:"wait_event_type" json:"wait_event_type"`
	WaitEvent       string  `csv:"wait_event" header:"wait_event" json:"wait_event"`
	Username        string  `csv:"username" header:"username" json:"username"`
	ApplicationName string  `csv:"application_name" header:"application_name" json:"application_name"`
	DatabaseName    string  `csv:"-" json:"database,omitempty"`
	ClientAddr      string  `csv:"client_addr" header:"client_addr" json:"client_addr"`
	QueryText       string  `csv:"query_text" header:"query_text" json:"query_text"`
	BlockedBy       []int   `csv:"blocked_by" header:"blocked_by" json:"blocked_by"`
	QueryID         *string `csv:"query_id" header:"query_id" json:"query_id"`
	TransactionID   *string `csv:"transaction_id" header:"transaction_id" json:"transaction_id"`
	ConnectionID    *string `csv:"connection_id" header:"connection_id" json:"connection_id"`
}

func toPrintableList(list live.ConnectionList, topology ListTopology) printableList {
	out := printableList{
		DatabaseKind: list.DatabaseKind,
		CapturedAt:   list.CapturedAt,
		Instances:    toPrintableInstances(list.Instances),
		Connections:  toPrintableConnections(list),
	}
	if !topology.isEmpty() {
		out.Topology = &topology
	}
	return out
}

func toPrintableInstances(instanceList []live.InstanceMeta) []printableInstance {
	instances := make([]printableInstance, 0, len(instanceList))
	for _, instance := range instanceList {
		instances = append(instances, printableInstance{
			ID:    instance.ID,
			Role:  instance.Role,
			Error: instance.Error,
		})
	}
	return instances
}

func (p printableList) MarshalCSVValue() interface{} {
	if p.Topology != nil {
		return p.connectionsWithTopology()
	}
	if p.DatabaseKind == live.DatabaseKindMySQL || p.hasDatabaseName() {
		return p.connectionsWithDatabase()
	}
	return p.Connections
}

func (p printableList) hasDatabaseName() bool {
	for _, conn := range p.Connections {
		if conn.DatabaseName != "" {
			return true
		}
	}
	return false
}

func toPrintableConnections(list live.ConnectionList) []printableConnection {
	connections := make([]printableConnection, 0, len(list.Connections))
	for _, conn := range list.Connections {
		connections = append(connections, printableConnection{
			PID:             conn.PID,
			Instance:        conn.Instance,
			InstanceRole:    conn.InstanceRole,
			State:           conn.State,
			DurationMS:      conn.Duration.Milliseconds(),
			WaitEventType:   conn.WaitEventType,
			WaitEvent:       conn.WaitEvent,
			Username:        conn.Username,
			ApplicationName: conn.ApplicationName,
			DatabaseName:    conn.DatabaseName,
			ClientAddr:      conn.ClientAddr,
			QueryText:       conn.QueryText,
			BlockedBy:       printableBlockedBy(conn.BlockedBy),
			QueryID:         conn.QueryID,
			TransactionID:   conn.TransactionID,
			ConnectionID:    conn.ConnectionID,
		})
	}
	return connections
}

func printableBlockedBy(blockedBy []int) []int {
	// Normalize an absent blocker set to [] so the agent JSON always sees an
	// array for blocked_by, consistent with instances. The wire field is
	// omitempty, so an unblocked connection decodes to a nil slice.
	if blockedBy == nil {
		return []int{}
	}
	return blockedBy
}

func printHumanConnectionList(out io.Writer, list live.ConnectionList, topology ListTopology) {
	fmt.Fprintf(out, "captured_at: %s\n", list.CapturedAt.Format(time.RFC3339))
	vitess := list.DatabaseKind == live.DatabaseKindMySQL || !topology.isEmpty()
	if !topology.isEmpty() {
		fmt.Fprintln(out, "topology:")
		fmt.Fprintf(out, "  keyspace: %s\n", topology.Keyspace)
		fmt.Fprintf(out, "  shard: %s\n", topology.Shard)
		fmt.Fprintf(out, "  tablet: %s\n", topology.Tablet)
	}
	if warning := unreachableInstanceWarning(list.Instances); warning != "" {
		fmt.Fprintln(out, warning)
	}

	if len(list.Connections) == 0 {
		fmt.Fprintln(out, "No live connections found.")
		return
	}

	for i, conn := range list.Connections {
		if i > 0 {
			fmt.Fprintln(out)
		}

		fmt.Fprintf(out, "*************************** %d. row ***************************\n", i+1)
		fields := conn.HumanFields()
		if vitess {
			fields = vitessHumanFields(conn)
		}
		for _, field := range fields {
			writeHumanField(out, field[0], field[1])
		}
		if conn.DatabaseName != "" {
			writeHumanField(out, "database", conn.DatabaseName)
		}
		fmt.Fprintln(out, "query:")
		fmt.Fprintln(out, conn.QueryText)
	}
}

func vitessHumanFields(conn live.Connection) [][2]string {
	fields := conn.HumanFields()
	out := make([][2]string, 0, len(fields)-1)
	for _, field := range fields {
		switch field[0] {
		case "instance":
			out = append(out, [2]string{"tablet", field[1]})
		case "role":
		default:
			out = append(out, field)
		}
	}
	return out
}

func writeHumanField(out io.Writer, name, value string) {
	fmt.Fprintf(out, "%-16s %s\n", name+":", value)
}

func unreachableInstanceWarning(instances []live.InstanceMeta) string {
	var unreachable []string
	for _, instance := range instances {
		if instance.Error != "" {
			unreachable = append(unreachable, instance.ID)
		}
	}
	if len(unreachable) == 0 {
		return ""
	}
	return "warning: partial results, unreachable instances: " + strings.Join(unreachable, ", ")
}

func validateInstanceFilter(list live.ConnectionList, filter connectionFilter) error {
	if filter.instance == "" {
		return nil
	}

	valid := make([]string, 0, len(list.Instances))
	for _, instance := range list.Instances {
		valid = append(valid, instance.ID)
		if instance.ID == filter.instance {
			return nil
		}
	}

	return &live.UnknownInstanceError{Instance: filter.instance, Valid: valid}
}

// sortListForDisplay orders Vitess processlist rows longest-running first so
// one-shot output matches the TUI's duration ordering. Postgres lists keep the
// server's transaction-start ordering.
func sortListForDisplay(list *live.ConnectionList) {
	if list.DatabaseKind != live.DatabaseKindMySQL {
		return
	}
	live.SortConnections(list.Connections, live.SortByDuration)
}
