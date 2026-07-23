package inspect

import (
	"regexp"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
)

var checkNameRE = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// TestCheckCatalog validates the invariants every check must hold: read-only,
// bounded SQL, an implementation or an alternative hint per engine, and
// well-formed metadata.
func TestCheckCatalog(t *testing.T) {
	c := qt.New(t)

	seen := map[string]bool{}
	for _, chk := range checks {
		c.Assert(checkNameRE.MatchString(chk.Name), qt.IsTrue, qt.Commentf("check name %q must be kebab-case", chk.Name))
		c.Assert(seen[chk.Name], qt.IsFalse, qt.Commentf("duplicate check name %q", chk.Name))
		seen[chk.Name] = true

		c.Assert(chk.Short, qt.Not(qt.Equals), "", qt.Commentf("%s: missing Short", chk.Name))
		c.Assert(chk.EmptyMessage, qt.Not(qt.Equals), "", qt.Commentf("%s: missing EmptyMessage", chk.Name))
		c.Assert(chk.MySQL != nil || chk.Postgres != nil, qt.IsTrue,
			qt.Commentf("%s: no engine implementation", chk.Name))

		if chk.MySQL == nil {
			c.Assert(chk.MySQLHint, qt.Not(qt.Equals), "", qt.Commentf("%s: MySQL unsupported but no hint", chk.Name))
		}
		if chk.Postgres == nil {
			c.Assert(chk.PostgresHint, qt.Not(qt.Equals), "", qt.Commentf("%s: Postgres unsupported but no hint", chk.Name))
		}

		for engine, impl := range map[string]*engineSQL{"mysql": chk.MySQL, "postgres": chk.Postgres} {
			if impl == nil {
				continue
			}
			sql := strings.ToUpper(strings.TrimSpace(impl.SQL))
			c.Assert(strings.HasPrefix(sql, "SELECT") || strings.HasPrefix(sql, "WITH"), qt.IsTrue,
				qt.Commentf("%s/%s: diagnostic SQL must be a read-only SELECT", chk.Name, engine))
			c.Assert(strings.Contains(sql, "LIMIT"), qt.IsTrue,
				qt.Commentf("%s/%s: diagnostic SQL must bound its result set with LIMIT", chk.Name, engine))
			c.Assert(strings.Count(impl.SQL, ";"), qt.Equals, 1,
				qt.Commentf("%s/%s: diagnostic SQL must be a single statement", chk.Name, engine))
			if chk.Name == "table-sizes" && engine == "postgres" {
				c.Assert(strings.Count(impl.SQL, "pg_total_relation_size"), qt.Equals, 2)
				c.Assert(strings.Contains(impl.SQL, "pg_table_size"), qt.IsFalse)
			}
		}

		for _, step := range chk.NextSteps {
			c.Assert(strings.HasPrefix(step, "pscale "), qt.IsTrue,
				qt.Commentf("%s: next step %q must be a pscale command", chk.Name, step))
		}
	}
}

func TestFormatNextSteps(t *testing.T) {
	c := qt.New(t)

	steps := formatNextSteps([]string{
		"pscale insights queries <database> <branch> --sort totalTime",
		"pscale insights recommendations <database>",
	}, "myorg", "mydb", "main")

	c.Assert(steps, qt.DeepEquals, []string{
		"pscale insights queries mydb main --sort totalTime --org myorg --format json",
		"pscale insights recommendations mydb --org myorg --format json",
	})
}

func TestFormatValue(t *testing.T) {
	c := qt.New(t)

	c.Assert(formatValue(nil), qt.Equals, "")
	c.Assert(formatValue("plain"), qt.Equals, "plain")
	c.Assert(formatValue("multi\n  line\tsql"), qt.Equals, "multi line sql")
	c.Assert(formatValue(int64(42)), qt.Equals, "42")
	c.Assert(formatValue(1.5), qt.Equals, "1.5")
}
