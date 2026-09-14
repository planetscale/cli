package vtctld

import (
	"encoding/json"
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
			data:        `{"traffic_state":"All Reads Switched. All Writes Switched"}`,
			wantCommand: "pscale branch vtctld move-tables complete my-db my-branch --org my-org --workflow my-workflow --target-keyspace target-ks --keep-data=false --keep-routing-rules=false --dry-run --format json",
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
