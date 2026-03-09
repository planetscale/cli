package branch

import (
	"reflect"
	"testing"

	"github.com/planetscale/cli/internal/postgres"
)

func TestCalculateSummary(t *testing.T) {
	tests := []struct {
		name   string
		states []postgres.TableReplicationState
		want   *ImportStatusSummaryOutput
	}{
		{
			name:   "empty",
			states: nil,
			want:   &ImportStatusSummaryOutput{Total: 0},
		},
		{
			name: "all ready",
			states: []postgres.TableReplicationState{
				{SchemaName: "public", TableName: "a", State: "r"},
				{SchemaName: "public", TableName: "b", State: "r"},
			},
			want: &ImportStatusSummaryOutput{Total: 2, Ready: 2},
		},
		{
			name: "mixed states",
			states: []postgres.TableReplicationState{
				{State: "r"},
				{State: "r"},
				{State: "d"},
				{State: "i"},
			},
			want: &ImportStatusSummaryOutput{Total: 4, Ready: 2, Copying: 1, Initializing: 1},
		},
		{
			name: "copying states d f s",
			states: []postgres.TableReplicationState{
				{State: "d"},
				{State: "f"},
				{State: "s"},
			},
			want: &ImportStatusSummaryOutput{Total: 3, Copying: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateSummary(tt.states)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calculateSummary() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
