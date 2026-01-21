package database

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestParseColumnIncludes(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name        string
		columns     []string
		want        map[string]map[string]bool
		wantErr     bool
		errContains string
	}{
		{
			name:    "single table single column",
			columns: []string{"users:id"},
			want: map[string]map[string]bool{
				"users": {"id": true},
			},
		},
		{
			name:    "single table multiple columns",
			columns: []string{"users:id,name,email"},
			want: map[string]map[string]bool{
				"users": {"id": true, "name": true, "email": true},
			},
		},
		{
			name:    "multiple tables",
			columns: []string{"users:id,name", "orders:id,total"},
			want: map[string]map[string]bool{
				"users":  {"id": true, "name": true},
				"orders": {"id": true, "total": true},
			},
		},
		{
			name:    "columns with whitespace",
			columns: []string{"users: id , name , email "},
			want: map[string]map[string]bool{
				"users": {"id": true, "name": true, "email": true},
			},
		},
		{
			name:    "same table specified multiple times merges columns",
			columns: []string{"users:id", "users:name,email"},
			want: map[string]map[string]bool{
				"users": {"id": true, "name": true, "email": true},
			},
		},
		{
			name:    "empty input",
			columns: []string{},
			want:    map[string]map[string]bool{},
		},
		{
			name:        "missing colon",
			columns:     []string{"users-id,name"},
			wantErr:     true,
			errContains: "expected 'table:col1,col2' format",
		},
		{
			name:        "empty table name",
			columns:     []string{":id,name"},
			wantErr:     true,
			errContains: "table name cannot be empty",
		},
		{
			name:        "empty column list",
			columns:     []string{"users:"},
			wantErr:     true,
			errContains: "at least one column must be specified",
		},
		{
			name:        "only whitespace columns",
			columns:     []string{"users: , , "},
			wantErr:     true,
			errContains: "at least one column must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseColumnIncludes(tt.columns)

			if tt.wantErr {
				c.Assert(err, qt.IsNotNil)
				if tt.errContains != "" {
					c.Assert(err.Error(), qt.Contains, tt.errContains)
				}
				return
			}

			c.Assert(err, qt.IsNil)
			c.Assert(got, qt.DeepEquals, tt.want)
		})
	}
}
