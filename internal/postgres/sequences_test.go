package postgres

import (
	"testing"
)

func TestEscapeSequenceRegclass(t *testing.T) {
	tests := []struct {
		name     string
		schema   string
		sequence string
		want     string
	}{
		{
			name:     "simple names",
			schema:   "public",
			sequence: "users_id_seq",
			want:     "public.users_id_seq",
		},
		{
			name:     "schema with single quote",
			schema:   "my'schema",
			sequence: "seq",
			want:     "my''schema.seq",
		},
		{
			name:     "sequence with single quote",
			schema:   "public",
			sequence: "user's_seq",
			want:     "public.user''s_seq",
		},
		{
			name:     "both with single quotes",
			schema:   "test'schema",
			sequence: "test'seq",
			want:     "test''schema.test''seq",
		},
		{
			name:     "multiple single quotes",
			schema:   "a'b'c",
			sequence: "x'y'z",
			want:     "a''b''c.x''y''z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeSequenceRegclass(tt.schema, tt.sequence)
			if got != tt.want {
				t.Errorf("escapeSequenceRegclass() = %v, want %v", got, tt.want)
			}
		})
	}
}
