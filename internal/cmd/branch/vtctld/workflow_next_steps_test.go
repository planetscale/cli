package vtctld

import (
	"encoding/json"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestMoveTablesStatusNextSteps(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		wantCommand string
		wantSteps   int
	}{
		{
			name:        "copying",
			data:        `{"table_copy_state":{"customers":{"rows_copied":10}},"traffic_state":"Reads Not Switched. Writes Not Switched"}`,
			wantCommand: "pscale branch vtctld move-tables status my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --format json",
			wantSteps:   1,
		},
		{
			name:        "running",
			data:        `{"table_copy_state":{},"shard_streams":{"target/-":{"streams":[{"status":"Running"}]}},"traffic_state":"Reads Not Switched. Writes Not Switched"}`,
			wantCommand: "pscale branch vtctld vdiff create my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --format json",
			wantSteps:   2,
		},
		{
			name:        "reads switched",
			data:        `{"traffic_state":"All Reads Switched. Writes Not Switched"}`,
			wantCommand: "pscale branch vtctld move-tables switch-traffic my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --tablet-types PRIMARY --format json",
			wantSteps:   1,
		},
		{
			name:        "all traffic switched",
			data:        `{"traffic_state":"All Reads Switched. Writes Switched"}`,
			wantCommand: "pscale branch vtctld move-tables complete my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --keep-data=false --keep-routing-rules=false --dry-run --format json",
			wantSteps:   1,
		},
		{
			name:        "writes switched but reads not",
			data:        `{"traffic_state":"Reads Not Switched. Writes Switched"}`,
			wantCommand: "pscale branch vtctld move-tables switch-traffic my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --tablet-types REPLICA,RDONLY --format json",
			wantSteps:   1,
		},
		{
			name:        "traffic not created",
			data:        `{"traffic_state":"Not Created"}`,
			wantCommand: "pscale branch vtctld move-tables status my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --format json",
			wantSteps:   1,
		},
		{
			name:        "unrecognized traffic state",
			data:        `{"traffic_state":"Something Vitess Has Not Told Us About"}`,
			wantCommand: "pscale branch vtctld move-tables status my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --format json",
			wantSteps:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			steps := moveTablesStatusNextSteps(
				json.RawMessage(tt.data),
				"my-org",
				"my-db",
				"my-branch",
				"my-workflow",
				"target-ks",
			)

			c.Assert(steps, qt.HasLen, tt.wantSteps)
			c.Assert(steps[0].Command, qt.Equals, tt.wantCommand)
		})
	}
}

func TestMoveTablesRunningOffersReplicaSwitchAfterVDiff(t *testing.T) {
	c := qt.New(t)

	steps := moveTablesStatusNextSteps(
		json.RawMessage(`{"shard_streams":{"target/-":{"streams":[{"status":"Running"}]}},"traffic_state":"Reads Not Switched. Writes Not Switched"}`),
		"my-org",
		"my-db",
		"my-branch",
		"my-workflow",
		"target-ks",
	)

	c.Assert(steps, qt.HasLen, 2)
	c.Assert(steps[1].Command, qt.Equals, "pscale branch vtctld move-tables switch-traffic my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --tablet-types REPLICA,RDONLY --format json")
	c.Assert(steps[1].Reason, qt.Equals, "Alternatively, skip VDiff and switch replica traffic directly")
}

// The constants must match the traffic states the API reports verbatim, aside
// from case.
func TestTrafficStateConstantsMatchAPI(t *testing.T) {
	c := qt.New(t)

	apiStates := []string{
		"Not Created",
		"Reads Not Switched. Writes Not Switched",
		"All Reads Switched. Writes Not Switched",
		"Reads Not Switched. Writes Switched",
		"All Reads Switched. Writes Switched",
	}
	cliStates := []string{
		trafficStateNotCreated,
		trafficStateNotSwitched,
		trafficStateReadsSwitched,
		trafficStateWritesSwitched,
		trafficStateAllSwitched,
	}

	for i, state := range apiStates {
		c.Assert(strings.ToLower(state), qt.Equals, cliStates[i])
	}
}

func TestWithNextStepsPreservesResponseShape(t *testing.T) {
	c := qt.New(t)

	steps := []workflowNextStep{{Command: "pscale ...", Reason: "check status"}}

	// Fields keep the order the API sent them in, and next_steps is appended
	// last rather than sorted into the middle of the response.
	enriched, err := withNextSteps(json.RawMessage(`{"workflow":"wf","traffic_state":"Not Switched","alpha":1}`), steps)
	c.Assert(err, qt.IsNil)
	c.Assert(string(enriched), qt.Equals, `{"workflow":"wf","traffic_state":"Not Switched","alpha":1,"next_steps":[{"command":"pscale ...","reason":"check status"}]}`)

	empty, err := withNextSteps(json.RawMessage(`{}`), steps)
	c.Assert(err, qt.IsNil)
	c.Assert(string(empty), qt.Equals, `{"next_steps":[{"command":"pscale ...","reason":"check status"}]}`)

	// Values are copied verbatim, so deep nesting and large integers that
	// would lose precision through a decode/encode round trip are unaffected.
	nested, err := withNextSteps(json.RawMessage(`{"rows_copied":90071992547409929,"shard_streams":{"ks/-":{"streams":[{"id":1}]}}}`), steps)
	c.Assert(err, qt.IsNil)
	c.Assert(string(nested), qt.Contains, `"rows_copied":90071992547409929`)
	c.Assert(string(nested), qt.Contains, `"shard_streams":{"ks/-":{"streams":[{"id":1}]}}`)
}

func TestWithNextStepsLeavesUnexpectedPayloadsAlone(t *testing.T) {
	c := qt.New(t)

	steps := []workflowNextStep{{Command: "pscale ...", Reason: "check status"}}

	for _, data := range []string{`null`, `[]`, `"a string"`, `12`, `not json`} {
		enriched, err := withNextSteps(json.RawMessage(data), steps)
		c.Assert(err, qt.IsNil)
		c.Assert(string(enriched), qt.Equals, data)
	}

	// An API that starts returning its own next_steps wins over ours.
	existing := `{"next_steps":["do the thing"]}`
	enriched, err := withNextSteps(json.RawMessage(existing), steps)
	c.Assert(err, qt.IsNil)
	c.Assert(string(enriched), qt.Equals, existing)

	// No steps to add means the response is returned byte for byte.
	unchanged, err := withNextSteps(json.RawMessage(`{"workflow":"wf"}`), nil)
	c.Assert(err, qt.IsNil)
	c.Assert(string(unchanged), qt.Equals, `{"workflow":"wf"}`)
}

func TestVDiffNextSteps(t *testing.T) {
	c := qt.New(t)

	createSteps := vdiffCreateNextSteps(
		json.RawMessage(`{"uuid":"abc-123"}`),
		"my-org",
		"my-db",
		"my-branch",
		"my-workflow",
		"target-ks",
	)
	c.Assert(createSteps, qt.DeepEquals, []workflowNextStep{{
		Command: "pscale branch vtctld vdiff show my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --uuid abc-123 --format json",
		Reason:  "Check VDiff progress",
	}})

	completedSteps := vdiffShowNextSteps(
		json.RawMessage(`{"summary":{"state":"STATE_COMPLETED","has_mismatch":false}}`),
		"my-org",
		"my-db",
		"my-branch",
		"my-workflow",
		"target-ks",
		"abc-123",
	)
	c.Assert(completedSteps, qt.DeepEquals, []workflowNextStep{{
		Command: "pscale branch vtctld move-tables switch-traffic my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --tablet-types REPLICA,RDONLY --format json",
		Reason:  "Switch replica traffic to the target keyspace",
	}})

	mismatchSteps := vdiffShowNextSteps(
		json.RawMessage(`{"summary":{"state":"STATE_COMPLETED","has_mismatch":true}}`),
		"my-org",
		"my-db",
		"my-branch",
		"my-workflow",
		"target-ks",
		"abc-123",
	)
	c.Assert(mismatchSteps, qt.DeepEquals, []workflowNextStep{{
		Reason: "VDiff found mismatches; inspect the report and resolve them before switching traffic",
	}})
}
