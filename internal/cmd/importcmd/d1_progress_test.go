package importcmd

import (
	"testing"

	"github.com/planetscale/cli/internal/import/d1"
)

func TestFormatProgressMessage(t *testing.T) {
	tests := []struct {
		name string
		in   d1.ImportProgress
		want string
	}{
		{
			name: "import sqlite staging",
			in:   d1.ImportProgress{Stage: d1.ImportStageSQLiteStaging},
			want: "Staging SQLite database from export...",
		},
		{
			name: "import pgloader table",
			in: d1.ImportProgress{
				Stage:   d1.ImportStagePgloader,
				Current: 3,
				Total:   19,
				Detail:  "team_members",
			},
			want: "Loading table 3/19: team_members",
		},
		{
			name: "verify row counts",
			in: d1.ImportProgress{
				Stage:   d1.VerifyStageRowCounts,
				Current: 2,
				Total:   19,
				Detail:  "users (postgres)",
			},
			want: "Counting rows 2/19: users (postgres)",
		},
		{
			name: "verify sequences",
			in:   d1.ImportProgress{Stage: d1.VerifyStageSequences},
			want: "Checking identity sequences...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatProgressMessage(tt.in); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
