package connections

import (
	"testing"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestValidateEngineFlagsForNeki(t *testing.T) {
	tests := []struct {
		name    string
		filter  ConnectionFilter
		target  ConnectionTarget
		wantErr string
	}{
		{name: "no flags"},
		{name: "vitess target", target: ConnectionTarget{Keyspace: "commerce"}, wantErr: "--keyspace/--shard are only supported for Vitess databases"},
		{name: "instance filter", filter: ConnectionFilter{Instance: "router-a"}, wantErr: "--instance/--role are only supported for Postgres databases"},
		{name: "role filter", filter: ConnectionFilter{Role: "primary"}, wantErr: "--instance/--role are only supported for Postgres databases"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			err := ValidateEngineFlags(ps.DatabaseEngineNeki, tt.filter, tt.target)

			if tt.wantErr == "" {
				c.Assert(err, qt.IsNil)
				return
			}
			c.Assert(err, qt.ErrorMatches, tt.wantErr)
		})
	}
}
