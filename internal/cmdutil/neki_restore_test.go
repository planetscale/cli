package cmdutil

import (
	"testing"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestParseConfigurationProfileSizes(t *testing.T) {
	c := qt.New(t)

	replicas := 2
	zero := 0

	got, err := ParseConfigurationProfileSizes([]string{
		"name=default,cluster-size=PS-40,replicas=2",
		"name=analytics, replicas = 0",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(got, qt.DeepEquals, []ps.ConfigurationProfileSize{
		{Name: "default", ClusterSize: "PS_40", Replicas: &replicas},
		{Name: "analytics", Replicas: &zero},
	})
}

func TestParseConfigurationProfileSizesErrors(t *testing.T) {
	c := qt.New(t)

	_, err := ParseConfigurationProfileSizes([]string{"cluster-size=PS_40"})
	c.Assert(err, qt.ErrorMatches, `.*missing required field name`)

	_, err = ParseConfigurationProfileSizes([]string{"name=default"})
	c.Assert(err, qt.ErrorMatches, `.*must set cluster-size and/or replicas`)

	_, err = ParseConfigurationProfileSizes([]string{"name=default,replicas=two"})
	c.Assert(err, qt.ErrorMatches, `.*replicas must be an integer`)

	_, err = ParseConfigurationProfileSizes([]string{"name=default,cluster-size=PS_10", "name=default,replicas=1"})
	c.Assert(err, qt.ErrorMatches, `duplicate --config-profile name "default"`)

	_, err = ParseConfigurationProfileSizes([]string{"name=default,metal=true"})
	c.Assert(err, qt.ErrorMatches, `unknown --config-profile field "metal".*`)
}

func TestParseRouterSizes(t *testing.T) {
	c := qt.New(t)

	replicas := 1
	got, err := ParseRouterSizes([]string{"name=default,size=NKR-20,replicas-per-cell=1"})
	c.Assert(err, qt.IsNil)
	c.Assert(got, qt.DeepEquals, []ps.RouterSize{
		{Name: "default", RouterSize: "NKR_20", ReplicasPerCell: &replicas},
	})
}

func TestParseRouterSizesErrors(t *testing.T) {
	c := qt.New(t)

	_, err := ParseRouterSizes([]string{"size=NKR_20"})
	c.Assert(err, qt.ErrorMatches, `.*missing required field name`)

	_, err = ParseRouterSizes([]string{"name=default"})
	c.Assert(err, qt.ErrorMatches, `.*must set size and/or replicas-per-cell`)

	_, err = ParseRouterSizes([]string{"name=default,replicas-per-cell=x"})
	c.Assert(err, qt.ErrorMatches, `.*replicas-per-cell must be an integer`)

	_, err = ParseRouterSizes([]string{"name=default,size=NKR_10", "name=default,replicas-per-cell=1"})
	c.Assert(err, qt.ErrorMatches, `duplicate --router name "default"`)
}

func TestEnsureNekiRestoreSizing(t *testing.T) {
	c := qt.New(t)

	c.Assert(EnsureNekiRestoreSizing(ps.DatabaseEngineMySQL, true, false, false), qt.IsNil)
	c.Assert(EnsureNekiRestoreSizing(ps.DatabaseEngineNeki, true, false, true), qt.IsNil)

	c.Assert(EnsureNekiRestoreSizing(ps.DatabaseEnginePostgres, true, false, true), qt.ErrorMatches, `.*only supported for Neki backup restores`)
	c.Assert(EnsureNekiRestoreSizing(ps.DatabaseEngineNeki, false, false, true), qt.ErrorMatches, `.*can only be used when restoring a backup`)
	c.Assert(EnsureNekiRestoreSizing(ps.DatabaseEngineNeki, true, true, true), qt.ErrorMatches, `.*not supported with --seed-data`)
}
