package branch

import (
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
)

func TestBranchCmdGroupsCommandsByEngine(t *testing.T) {
	c := qt.New(t)

	cmd := BranchCmd(&cmdutil.Helper{Config: &config.Config{}})

	groups := map[string]string{}
	for _, group := range cmd.Groups() {
		groups[group.ID] = group.Title
	}
	c.Assert(groups[branchGroupDatabase], qt.Contains, "MySQL, Postgres, and Neki branch commands:")
	c.Assert(groups[branchGroupMySQLPostgres], qt.Contains, "MySQL and Postgres:")
	c.Assert(groups[branchGroupVitess], qt.Contains, "Vitess/MySQL-specific:")
	c.Assert(groups[branchGroupPostgres], qt.Contains, "Postgres-specific:")
	c.Assert(groups[branchGroupPostgresNeki], qt.Contains, "Postgres and Neki:")
	c.Assert(groups[branchGroupNeki], qt.Contains, "Neki-specific:")

	got := map[string]string{}
	for _, command := range cmd.Commands() {
		got[command.Name()] = command.GroupID
	}

	c.Assert(got["create"], qt.Equals, branchGroupDatabase)
	c.Assert(got["connections"], qt.Equals, branchGroupMySQLPostgres)
	c.Assert(got["lint"], qt.Equals, branchGroupVitess)
	c.Assert(got["routing-rules"], qt.Equals, branchGroupVitess)
	c.Assert(got["resize"], qt.Equals, branchGroupPostgres)
	c.Assert(got["parameters"], qt.Equals, branchGroupPostgres)
	c.Assert(got["maintenance"], qt.Equals, branchGroupPostgresNeki)
	c.Assert(got["data-topology"], qt.Equals, branchGroupNeki)
	c.Assert(got["admin"], qt.Equals, branchGroupNeki)
	c.Assert(got["config-profile"], qt.Equals, branchGroupNeki)
	c.Assert(got["router"], qt.Equals, branchGroupNeki)
	c.Assert(got["shard"], qt.Equals, branchGroupNeki)
	c.Assert(got["sidecar"], qt.Equals, branchGroupNeki)
}
