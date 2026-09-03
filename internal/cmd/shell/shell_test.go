package shell

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

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

