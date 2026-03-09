package postgres

import (
	"testing"
	"time"
)

func TestTableStateDescription(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  string
	}{
		{
			name:  "initializing state",
			state: "i",
			want:  "initializing",
		},
		{
			name:  "copying data state",
			state: "d",
			want:  "copying data",
		},
		{
			name:  "finished copy state",
			state: "f",
			want:  "finished copy",
		},
		{
			name:  "synchronized state",
			state: "s",
			want:  "synchronized",
		},
		{
			name:  "ready state",
			state: "r",
			want:  "ready",
		},
		{
			name:  "unknown state",
			state: "x",
			want:  "unknown",
		},
		{
			name:  "empty state",
			state: "",
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TableStateDescription(tt.state)
			if got != tt.want {
				t.Errorf("TableStateDescription(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestPublicationOptions(t *testing.T) {
	// Test basic struct initialization
	opts := PublicationOptions{
		Name:      "test_pub",
		AllTables: true,
		Schemas:   []string{"public", "app"},
	}

	if opts.Name != "test_pub" {
		t.Errorf("Name = %v, want test_pub", opts.Name)
	}
	if !opts.AllTables {
		t.Error("AllTables should be true")
	}
	if len(opts.Schemas) != 2 {
		t.Errorf("Schemas length = %v, want 2", len(opts.Schemas))
	}
}

func TestSubscriptionOptions(t *testing.T) {
	// Test basic struct initialization
	opts := SubscriptionOptions{
		Name:             "test_sub",
		SourceConnString: "postgresql://localhost:5432/db",
		PublicationName:  "test_pub",
		CopyData:         true,
		CreateSlot:       false,
		SlotName:         "test_slot",
		Enabled:          true,
	}

	if opts.Name != "test_sub" {
		t.Errorf("Name = %v, want test_sub", opts.Name)
	}
	if !opts.CopyData {
		t.Error("CopyData should be true")
	}
	if opts.CreateSlot {
		t.Error("CreateSlot should be false")
	}
	if !opts.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestSubscriptionStatus(t *testing.T) {
	// Test basic struct initialization with time fields
	now := time.Now()
	lag := 5 * time.Second

	status := SubscriptionStatus{
		Name:            "test_sub",
		Enabled:         true,
		SlotName:        "test_slot",
		PublicationName: "test_pub",
		ReceivedLSN:     "0/12345678",
		LatestEndLSN:    "0/12345679",
		LastMsgSendTime: &now,
		LastMsgRecvTime: &now,
		LatestEndTime:   &now,
		ReplicationLag:  &lag,
	}

	if status.Name != "test_sub" {
		t.Errorf("Name = %v, want test_sub", status.Name)
	}
	if !status.Enabled {
		t.Error("Enabled should be true")
	}
	if status.ReplicationLag == nil {
		t.Error("ReplicationLag should not be nil")
	}
	if *status.ReplicationLag != lag {
		t.Errorf("ReplicationLag = %v, want %v", *status.ReplicationLag, lag)
	}
}

func TestTableReplicationState(t *testing.T) {
	// Test basic struct initialization
	state := TableReplicationState{
		SchemaName: "public",
		TableName:  "users",
		State:      "r",
		LSN:        "0/12345678",
	}

	if state.SchemaName != "public" {
		t.Errorf("SchemaName = %v, want public", state.SchemaName)
	}
	if state.TableName != "users" {
		t.Errorf("TableName = %v, want users", state.TableName)
	}
	if state.State != "r" {
		t.Errorf("State = %v, want r", state.State)
	}

	// Test with TableStateDescription
	desc := TableStateDescription(state.State)
	if desc != "ready" {
		t.Errorf("TableStateDescription(%q) = %q, want ready", state.State, desc)
	}
}

func TestPreflightCheck(t *testing.T) {
	// Test basic struct initialization
	check := PreflightCheck{
		WALLevel:                 "logical",
		WALLevelOK:               true,
		MaxReplicationSlots:      10,
		SlotsAvailable:           5,
		HasReplicationPermission: true,
		Extensions:               []string{"pg_stat_statements", "uuid-ossp"},
	}

	if check.WALLevel != "logical" {
		t.Errorf("WALLevel = %v, want logical", check.WALLevel)
	}
	if !check.WALLevelOK {
		t.Error("WALLevelOK should be true")
	}
	if check.SlotsAvailable != 5 {
		t.Errorf("SlotsAvailable = %v, want 5", check.SlotsAvailable)
	}
	if len(check.Extensions) != 2 {
		t.Errorf("Extensions length = %v, want 2", len(check.Extensions))
	}
}
