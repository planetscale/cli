package insights

import (
	"testing"

	ps "github.com/planetscale/cli/internal/planetscale"

	qt "github.com/frankban/quicktest"
)

func TestResolveTagIDs(t *testing.T) {
	c := qt.New(t)

	available := []*ps.QueryTag{
		{ID: "Sapp", Name: "app", Source: "sql"},
		{ID: "Busername", Name: "username", Source: "system"},
		{ID: "Scontroller", Name: "controller", Source: "sql"},
		{ID: "Bcontroller", Name: "controller", Source: "system"},
		{ID: "SStatus", Name: "Status", Source: "sql"},
	}

	c.Run("friendly name", func(c *qt.C) {
		ids, err := resolveTagIDs(available, []string{"app", "username"})
		c.Assert(err, qt.IsNil)
		c.Assert(ids, qt.DeepEquals, []string{"Sapp", "Busername"})
	})

	c.Run("friendly name starting with S", func(c *qt.C) {
		ids, err := resolveTagIDs(available, []string{"Status"})
		c.Assert(err, qt.IsNil)
		c.Assert(ids, qt.DeepEquals, []string{"SStatus"})
	})

	c.Run("prefixed id", func(c *qt.C) {
		ids, err := resolveTagIDs(available, []string{"Sapp"})
		c.Assert(err, qt.IsNil)
		c.Assert(ids, qt.DeepEquals, []string{"Sapp"})
	})

	c.Run("source qualified", func(c *qt.C) {
		ids, err := resolveTagIDs(available, []string{"sql:controller", "system:controller"})
		c.Assert(err, qt.IsNil)
		c.Assert(ids, qt.DeepEquals, []string{"Scontroller", "Bcontroller"})
	})

	c.Run("ambiguous name", func(c *qt.C) {
		_, err := resolveTagIDs(available, []string{"controller"})
		c.Assert(err, qt.ErrorMatches, `(?s).*matches multiple sources.*`)
	})

	c.Run("missing", func(c *qt.C) {
		_, err := resolveTagIDs(available, []string{"nope"})
		c.Assert(err, qt.ErrorMatches, `(?s).*not found.*`)
	})
}

func TestFormatDimensions(t *testing.T) {
	c := qt.New(t)
	got := formatDimensions(map[string]string{
		"Scontroller": "users",
		"Bcontroller": "admin",
	})
	c.Assert(got, qt.Equals, "sql:controller=users system:controller=admin")
}
