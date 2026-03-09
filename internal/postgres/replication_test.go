package postgres

import (
	"testing"
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

