package vtctld

import (
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
	if len(steps) == 0 {
		return p.PrettyPrintJSON(data)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	rawSteps, err := json.Marshal(steps)
	if err != nil {
		return err
	}
	payload["next_steps"] = rawSteps

	enriched, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.PrettyPrintJSON(enriched)
}

func printMoveTablesListJSON(p *printer.Printer, data json.RawMessage, org, database, branch string) error {
	var workflows []map[string]json.RawMessage
	if err := json.Unmarshal(data, &workflows); err != nil {
		return err
	}

	for _, workflow := range workflows {
		var name string
		var targetKeyspace string
		if err := json.Unmarshal(workflow["name"], &name); err != nil || name == "" {
			continue
		}
		if err := json.Unmarshal(workflow["target_keyspace"], &targetKeyspace); err != nil || targetKeyspace == "" {
			continue
		}

		steps, err := json.Marshal([]workflowNextStep{
			moveTablesStatusStep(org, database, branch, name, targetKeyspace, "Check workflow copy and traffic state"),
		})
		if err != nil {
			return err
		}
		workflow["next_steps"] = steps
	}

	enriched, err := json.Marshal(workflows)
	if err != nil {
		return err
	}
	return p.PrettyPrintJSON(enriched)
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

func moveTablesVDiffCreateStep(org, database, branch, workflow, targetKeyspace string) workflowNextStep {
	return workflowNextStep{
		Command: fmt.Sprintf(
			"pscale branch vtctld vdiff create %s %s --org %s --workflow %s --target-keyspace %s --format json",
			database, branch, org, workflow, targetKeyspace,
		),
		Reason: "Verify source and target data before switching traffic",
	}
}

func moveTablesSwitchReadsStep(org, database, branch, workflow, targetKeyspace string) workflowNextStep {
	return workflowNextStep{
		Command: moveTablesCommand(org, "switch-traffic", database, branch, workflow, targetKeyspace, "--tablet-types", "REPLICA,RDONLY"),
		Reason:  "Switch replica traffic to the target keyspace",
	}
}

func moveTablesSwitchReadsAlternativeStep(org, database, branch, workflow, targetKeyspace string) workflowNextStep {
	step := moveTablesSwitchReadsStep(org, database, branch, workflow, targetKeyspace)
	step.Reason = "Alternatively, skip VDiff and switch replica traffic directly"
	return step
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

	hasStreams, streamsNeedMonitoring := moveTablesStreamState(status)
	if len(status.TableCopyState) > 0 || streamsNeedMonitoring {
		return []workflowNextStep{
			moveTablesStatusStep(org, database, branch, workflow, targetKeyspace, "Copy or replication is still in progress; check status again"),
		}
	}

	switch strings.ToLower(status.TrafficState) {
	case "reads not switched. writes not switched":
		if !hasStreams {
			return []workflowNextStep{
				moveTablesStatusStep(org, database, branch, workflow, targetKeyspace, "Wait for workflow streams to start"),
			}
		}
		return []workflowNextStep{
			moveTablesVDiffCreateStep(org, database, branch, workflow, targetKeyspace),
			moveTablesSwitchReadsAlternativeStep(org, database, branch, workflow, targetKeyspace),
		}
	case "all reads switched. writes not switched":
		return []workflowNextStep{
			moveTablesSwitchPrimaryStep(org, database, branch, workflow, targetKeyspace),
		}
	case "reads not switched. writes switched":
		return []workflowNextStep{
			moveTablesSwitchReadsStep(org, database, branch, workflow, targetKeyspace),
		}
	case "all reads switched. all writes switched":
		return []workflowNextStep{
			moveTablesCompleteStep(org, database, branch, workflow, targetKeyspace),
		}
	default:
		return []workflowNextStep{
			moveTablesStatusStep(org, database, branch, workflow, targetKeyspace, "Check workflow copy and traffic state again"),
		}
	}
}

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
		return []workflowNextStep{moveTablesSwitchReadsStep(org, database, branch, workflow, targetKeyspace)}
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
