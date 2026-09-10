package shell

import (
	"testing"

	ps "github.com/planetscale/cli/internal/planetscale"

	qt "github.com/frankban/quicktest"
)

func TestShellUsername(t *testing.T) {
	c := qt.New(t)

	username, err := shellUsername("role-abc", false, "", ps.DatabaseEngineNeki)
	c.Assert(err, qt.IsNil)
	c.Assert(username, qt.Equals, "role-abc")

	username, err = shellUsername("role-abc", true, "", ps.DatabaseEnginePostgres)
	c.Assert(err, qt.IsNil)
	c.Assert(username, qt.Equals, "role-abc|replica")

	// Neki replica reads go through PGOPTIONS, not the username suffix.
	username, err = shellUsername("role-abc", true, "", ps.DatabaseEngineNeki)
	c.Assert(err, qt.IsNil)
	c.Assert(username, qt.Equals, "role-abc")

	username, err = shellUsername("role-abc", false, "analytics", ps.DatabaseEngineNeki)
	c.Assert(err, qt.IsNil)
	c.Assert(username, qt.Equals, "role-abc|analytics")

	username, err = shellUsername("role-abc", true, "analytics", ps.DatabaseEngineNeki)
	c.Assert(err, qt.IsNil)
	c.Assert(username, qt.Equals, "role-abc|analytics")

	_, err = shellUsername("role-abc", false, "analytics", ps.DatabaseEnginePostgres)
	c.Assert(err, qt.ErrorMatches, "--router is only supported for Neki databases")
}

func TestShellPgOptions(t *testing.T) {
	c := qt.New(t)

	c.Assert(shellPgOptions(true, ps.DatabaseEngineNeki), qt.Equals, nekiReplicaOptions)
	c.Assert(shellPgOptions(false, ps.DatabaseEngineNeki), qt.Equals, "")
	c.Assert(shellPgOptions(true, ps.DatabaseEnginePostgres), qt.Equals, "")
}

func TestPostgresPsqlArgs_DefaultDBName(t *testing.T) {
	c := qt.New(t)

	args := postgresPsqlArgs("db.example.com", "5432", "test-user", "", "prod/my-branch> ")

	c.Assert(args, qt.DeepEquals, []string{
		"-h", "db.example.com",
		"-p", "5432",
		"-U", "test-user",
		"-d", "postgres",
		"-v", "PROMPT1=prod/my-branch> ",
	})
}

func TestPostgresPsqlArgs_CustomDBName(t *testing.T) {
	c := qt.New(t)

	args := postgresPsqlArgs("db.example.com", "5432", "test-user", "my_db", "prod/my-branch> ")

	c.Assert(args, qt.DeepEquals, []string{
		"-h", "db.example.com",
		"-p", "5432",
		"-U", "test-user",
		"-d", "my_db",
		"-v", "PROMPT1=prod/my-branch> ",
	})
}
