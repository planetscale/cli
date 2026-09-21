package vtctld

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/printer"
)

type workflowNextStep struct {
	Command string `json:"command,omitempty"`
	Reason  string `json:"reason"`
}

type moveTablesStatus struct {
	TrafficState   string                     `json:"traffic_state"`
	TableCopyState map[string]json.RawMessage `json:"table_copy_state"`
	ShardStreams   map[string]struct {
		Streams []struct {
			Status string `json:"status"`
		} `json:"streams"`
	} `json:"shard_streams"`
}

type vdiffResult struct {
	UUID    string `json:"uuid"`
	Summary struct {
		State       string `json:"state"`
		HasMismatch bool   `json:"has_mismatch"`
	} `json:"summary"`
}

func printWorkflowJSON(p *printer.Printer, data json.RawMessage, steps []workflowNextStep) error {
	enriched, err := withNextSteps(data, steps)
	if err != nil {
		return err
	}
	return p.PrettyPrintJSON(enriched)
}

func printMoveTablesListJSON(p *printer.Printer, data json.RawMessage, org, database, branch string) error {
	enriched, err := withMoveTablesListNextSteps(data, org, database, branch)
	if err != nil {
		return err
	}
	return p.PrettyPrintJSON(enriched)
}

func withMoveTablesListNextSteps(data json.RawMessage, org, database, branch string) (json.RawMessage, error) {
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
		return withMoveTablesWorkflowArrayNextSteps(data, org, database, branch)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return data, nil
	}
	workflows, ok := fields["workflows"]
	if !ok || !bytes.HasPrefix(bytes.TrimSpace(workflows), []byte("[")) {
		return data, nil
	}

	enriched, err := withMoveTablesWorkflowArrayNextSteps(workflows, org, database, branch)
	if err != nil {
		return nil, err
	}

	index := bytes.Index(data, workflows)
	if index < 0 {
		return data, nil
	}

	result := make([]byte, 0, len(data)+len(enriched)-len(workflows))
	result = append(result, data[:index]...)
	result = append(result, enriched...)
	result = append(result, data[index+len(workflows):]...)
	return result, nil
}

func withMoveTablesWorkflowArrayNextSteps(data json.RawMessage, org, database, branch string) (json.RawMessage, error) {
	var workflows []json.RawMessage
	if err := json.Unmarshal(data, &workflows); err != nil {
		return data, nil
	}

	enriched := make([][]byte, 0, len(workflows))
	for _, workflow := range workflows {
		name, targetKeyspace := moveTablesListEntryTarget(workflow)
		if name == "" || targetKeyspace == "" {
			enriched = append(enriched, workflow)
			continue
		}

		withSteps, err := withNextSteps(workflow, []workflowNextStep{
			moveTablesStatusStep(org, database, branch, name, targetKeyspace, "Check workflow copy and traffic state"),
		})
		if err != nil {
			return nil, err
		}
		enriched = append(enriched, withSteps)
	}

	var buf bytes.Buffer
	buf.WriteByte('[')
	buf.Write(bytes.Join(enriched, []byte(",")))
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

func moveTablesListEntryTarget(workflow json.RawMessage) (string, string) {
	var entry struct {
		Name           string `json:"name"`
		TargetKeyspace string `json:"target_keyspace"`
		Target         struct {
			Keyspace string `json:"keyspace"`
		} `json:"target"`
	}
	if err := json.Unmarshal(workflow, &entry); err != nil {
		return "", ""
	}
	if entry.TargetKeyspace == "" {
		entry.TargetKeyspace = entry.Target.Keyspace
	}
	return entry.Name, entry.TargetKeyspace
}

// withNextSteps adds a next_steps field to a JSON object, splicing it in rather
// than re-encoding so that every field the API returned, and the order it
// returned them in, survives untouched. Payloads that are not JSON objects, and
// those that already carry a next_steps field, are returned unchanged.
func withNextSteps(data json.RawMessage, steps []workflowNextStep) (json.RawMessage, error) {
	if len(steps) == 0 {
		return data, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return data, nil
	}
	if _, ok := fields["next_steps"]; ok {
		return data, nil
	}

	encoded, err := json.Marshal(steps)
	if err != nil {
		return nil, err
	}

	object := bytes.TrimSpace(data)
	body := bytes.TrimSpace(object[1 : len(object)-1])

	var buf bytes.Buffer
	buf.WriteByte('{')
	if len(body) > 0 {
		buf.Write(body)
		buf.WriteByte(',')
	}
	buf.WriteString(`"next_steps":`)
	buf.Write(encoded)
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func moveTablesCommand(org, action, database, branch, workflow, targetKeyspace string, flags ...string) string {
	parts := []string{
		"pscale", "branch", "vtctld", "move-tables", action, database, branch,
		"--org", org,
		"--workflow", workflow,
		"--target-keyspace", targetKeyspace,
	}
	parts = append(parts, flags...)
	parts = append(parts, "--format", "json")
	return strings.Join(parts, " ")
}

func moveTablesStatusStep(org, database, branch, workflow, targetKeyspace, reason string) workflowNextStep {
	return workflowNextStep{
		Command: moveTablesCommand(org, "status", database, branch, workflow, targetKeyspace),
		Reason:  reason,
	}
}

func moveTablesStartStep(org, database, branch, workflow, targetKeyspace, reason string) workflowNextStep {
	return workflowNextStep{
		Command: moveTablesCommand(org, "start", database, branch, workflow, targetKeyspace),
		Reason:  reason,
	}
}

func moveTablesVDiffCreateStep(org, database, branch, workflow, targetKeyspace string) workflowNextStep {
	return workflowNextStep{
		Command: fmt.Sprintf(
			"pscale branch vtctld vdiff create %s %s --org %s --workflow %s --target-keyspace %s --format json",
			database, branch, org, workflow, targetKeyspace,
		),
		Reason: "Verify source and target data before switching traffic",
	}
}

func moveTablesSwitchReadsStep(org, database, branch, workflow, targetKeyspace, reason string) workflowNextStep {
	return workflowNextStep{
		Command: moveTablesCommand(org, "switch-traffic", database, branch, workflow, targetKeyspace, "--tablet-types", "REPLICA,RDONLY"),
		Reason:  reason,
	}
}

func moveTablesSwitchPrimaryStep(org, database, branch, workflow, targetKeyspace string) workflowNextStep {
	return workflowNextStep{
		Command: moveTablesCommand(org, "switch-traffic", database, branch, workflow, targetKeyspace, "--tablet-types", "PRIMARY"),
		Reason:  "Switch primary traffic after validating replica traffic",
	}
}

func moveTablesCompleteStep(org, database, branch, workflow, targetKeyspace string) workflowNextStep {
	return workflowNextStep{
		Command: moveTablesCommand(
			org, "complete", database, branch, workflow, targetKeyspace,
			"--keep-data=false", "--keep-routing-rules=false", "--dry-run",
		),
		Reason: "Preview cleanup after all traffic has switched",
	}
}

func moveTablesStatusNextSteps(data json.RawMessage, org, database, branch, workflow, targetKeyspace string) []workflowNextStep {
	var status moveTablesStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil
	}

	if moveTablesStreamsAllStopped(status) {
		return []workflowNextStep{
			moveTablesStartStep(org, database, branch, workflow, targetKeyspace, "Resume the stopped workflow"),
		}
	}

	hasStreams, streamsNeedMonitoring := moveTablesStreamState(status)
	if len(status.TableCopyState) > 0 || streamsNeedMonitoring {
		return []workflowNextStep{
			moveTablesStatusStep(org, database, branch, workflow, targetKeyspace, "Copy or replication is still in progress; check status again"),
		}
	}

	switch strings.ToLower(strings.TrimSpace(status.TrafficState)) {
	case trafficStateAllSwitched:
		return []workflowNextStep{
			moveTablesCompleteStep(org, database, branch, workflow, targetKeyspace),
		}
	case trafficStateReadsSwitched:
		return []workflowNextStep{
			moveTablesSwitchPrimaryStep(org, database, branch, workflow, targetKeyspace),
		}
	case trafficStateWritesSwitched:
		return []workflowNextStep{
			moveTablesSwitchReadsStep(org, database, branch, workflow, targetKeyspace, "Switch replica traffic to the target keyspace"),
		}
	case trafficStateNotSwitched:
		if !hasStreams {
			return []workflowNextStep{
				moveTablesStatusStep(org, database, branch, workflow, targetKeyspace, "Wait for workflow streams to start"),
			}
		}
		return []workflowNextStep{
			moveTablesVDiffCreateStep(org, database, branch, workflow, targetKeyspace),
			moveTablesSwitchReadsStep(org, database, branch, workflow, targetKeyspace, "Alternatively, skip VDiff and switch replica traffic directly"),
		}
	case trafficStateNotCreated:
		return []workflowNextStep{
			moveTablesStatusStep(org, database, branch, workflow, targetKeyspace, "Workflow is not routing traffic yet; check status again"),
		}
	default:
		return []workflowNextStep{
			moveTablesStatusStep(org, database, branch, workflow, targetKeyspace, "Check workflow copy and traffic state again"),
		}
	}
}

// The traffic states a workflow reports, lowercased so the comparison is
// case-insensitive. Any other value falls through to suggesting another status
// check rather than guessing at a lifecycle step.
const (
	trafficStateNotCreated     = "not created"
	trafficStateNotSwitched    = "reads not switched. writes not switched"
	trafficStateReadsSwitched  = "all reads switched. writes not switched"
	trafficStateWritesSwitched = "reads not switched. writes switched"
	trafficStateAllSwitched    = "all reads switched. writes switched"
)

func moveTablesStreamState(status moveTablesStatus) (bool, bool) {
	hasStreams := false
	for _, shard := range status.ShardStreams {
		for _, stream := range shard.Streams {
			hasStreams = true
			if !strings.EqualFold(stream.Status, "Running") {
				return true, true
			}
		}
	}
	return hasStreams, false
}

func moveTablesStreamsAllStopped(status moveTablesStatus) bool {
	count := 0
	for _, shard := range status.ShardStreams {
		for _, stream := range shard.Streams {
			count++
			if !strings.EqualFold(stream.Status, "Stopped") {
				return false
			}
		}
	}
	return count > 0
}

func vdiffCreateNextSteps(data json.RawMessage, org, database, branch, workflow, targetKeyspace string) []workflowNextStep {
	var result vdiffResult
	if err := json.Unmarshal(data, &result); err != nil || result.UUID == "" {
		return nil
	}
	return []workflowNextStep{vdiffShowStep(org, database, branch, workflow, targetKeyspace, result.UUID, "Check VDiff progress")}
}

func vdiffShowNextSteps(data json.RawMessage, org, database, branch, workflow, targetKeyspace, uuid string) []workflowNextStep {
	var result vdiffResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}

	switch strings.ToUpper(result.Summary.State) {
	case "STATE_COMPLETED":
		if result.Summary.HasMismatch {
			return []workflowNextStep{{
				Reason: "VDiff found mismatches; inspect the report and resolve them before switching traffic",
			}}
		}
		return []workflowNextStep{
			moveTablesSwitchReadsStep(org, database, branch, workflow, targetKeyspace, "Switch replica traffic to the target keyspace"),
		}
	case "STATE_STOPPED":
		return []workflowNextStep{{
			Command: fmt.Sprintf(
				"pscale branch vtctld vdiff resume %s %s --org %s --workflow %s --target-keyspace %s --uuid %s --format json",
				database, branch, org, workflow, targetKeyspace, uuid,
			),
			Reason: "Resume the stopped VDiff",
		}}
	case "STATE_ERROR":
		return []workflowNextStep{{
			Reason: "VDiff failed; inspect summary.errors before retrying",
		}}
	default:
		return []workflowNextStep{vdiffShowStep(org, database, branch, workflow, targetKeyspace, uuid, "VDiff is still in progress; check status again")}
	}
}

func vdiffShowStep(org, database, branch, workflow, targetKeyspace, uuid, reason string) workflowNextStep {
	return workflowNextStep{
		Command: fmt.Sprintf(
			"pscale branch vtctld vdiff show %s %s --org %s --workflow %s --target-keyspace %s --uuid %s --format json",
			database, branch, org, workflow, targetKeyspace, uuid,
		),
		Reason: reason,
	}
}
