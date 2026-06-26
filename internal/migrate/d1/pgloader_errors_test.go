package d1

import "testing"

func TestPgloaderHadErrors(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name: "clean summary",
			output: `
| errors | rows | bytes | total time
|      0 |  100 |  1 kB | 1.000 s
`,
			want: false,
		},
		{
			name: "summary with errors",
			output: `
| errors | rows | bytes | total time
|      3 |   97 |  1 kB | 1.000 s
`,
			want: true,
		},
		{
			name:   "database error",
			output: "Database error 42501: must be owner of table users",
			want:   true,
		},
		{
			name:   "insufficient privilege",
			output: "INSUFFICIENT-PRIVILEGE disable triggers",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pgloaderHadErrors(tt.output); got != tt.want {
				t.Fatalf("pgloaderHadErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}
