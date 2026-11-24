package password

import (
	"testing"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestToConnectionTypeDesc(t *testing.T) {
	c := qt.New(t)

	c.Assert(toConnectionTypeDesc(&ps.DatabaseBranchPassword{}), qt.Equals, "Primary")
	c.Assert(toConnectionTypeDesc(&ps.DatabaseBranchPassword{Replica: true}), qt.Equals, "Replica")
	c.Assert(toConnectionTypeDesc(&ps.DatabaseBranchPassword{ReadOnlyRegion: true}), qt.Equals, "Read-only region")
	c.Assert(toConnectionTypeDesc(&ps.DatabaseBranchPassword{
		ReadOnlyRegion: true,
		Region:         ps.Region{Slug: "eu-west"},
	}), qt.Equals, "Read-only region (eu-west)")
}

func TestToPasswordClientTabletTargetingPolicy(t *testing.T) {
	c := qt.New(t)

	password := &ps.DatabaseBranchPassword{Replica: true, ClientTabletTargetingPolicy: "ignore"}

	c.Assert(toPassword(password).ClientTabletTargetingPolicy, qt.Equals, "ignore")
	c.Assert(toPasswordWithoutTTL(password).ClientTabletTargetingPolicy, qt.Equals, "ignore")
	c.Assert(toPasswordWithPlainText(password).ClientTabletTargetingPolicy, qt.Equals, "ignore")
}
