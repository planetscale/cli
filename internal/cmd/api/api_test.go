package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseField(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		init   map[string]interface{}
		want   map[string]interface{}
	}{
		{
			name: "fields only",
			fields: []string{
				"hello.world=1",
				"hello.monde=2",
				`salut="le monde"`,
			},
			want: map[string]interface{}{
				"hello": map[string]interface{}{
					"world": float64(1),
					"monde": float64(2),
				},
				"salut": "le monde",
			},
		},
		{
			name: "update from fields",
			init: map[string]interface{}{
				"hello": map[string]interface{}{
					"monde": float64(42),
				},
				"salut": "fred",
				"bye":   "ivon",
			},
			fields: []string{
				"hello.world=1",
				"hello.monde=2",
				`salut="le monde"`,
			},
			want: map[string]interface{}{
				"hello": map[string]interface{}{
					"world": float64(1),
					"monde": float64(2),
				},
				"salut": "le monde",
				"bye":   "ivon",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := parseFields(tt.init, tt.fields)
			require.NoError(t, err)
			require.Equal(t, tt.want, out)
		})
	}
}
