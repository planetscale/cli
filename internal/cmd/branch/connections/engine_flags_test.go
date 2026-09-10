package connections

import (
	"testing"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestValidateEngineFlagsRejectsNeki(t *testing.T) {
	c := qt.New(t)

	err := ValidateEngineFlags(ps.DatabaseEngineNeki, ConnectionFilter{}, ConnectionTarget{})

	c.Assert(err, qt.ErrorMatches, "connections is not supported for Neki databases")
}
