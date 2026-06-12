package tui

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestFormatQueryForDisplayFormatsCommonSingleLineQuery(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select id, account_id from invoices where account_id = $1 and status = $2 order by created_at desc limit 200")

	c.Assert(got, qt.DeepEquals, []string{
		"select id, account_id",
		"from invoices",
		"where account_id = $1",
		"  and status = $2",
		"order by created_at desc",
		"limit 200",
	})
}

func TestFormatQueryForDisplayFormatsJoinQuery(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select invoices.id, accounts.name from invoices join accounts on accounts.id = invoices.account_id where invoices.status = $1 order by invoices.created_at desc limit 100")

	c.Assert(got, qt.DeepEquals, []string{
		"select invoices.id, accounts.name",
		"from invoices",
		"join accounts on accounts.id = invoices.account_id",
		"where invoices.status = $1",
		"order by invoices.created_at desc",
		"limit 100",
	})
}

func TestFormatQueryForDisplayFormatsDMLBoundaries(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("update users set name = $1 where id = $2 returning id")

	c.Assert(got, qt.DeepEquals, []string{
		"update users",
		"set name = $1",
		"where id = $2",
		"returning id",
	})
}

func TestFormatQueryForDisplayFormatsDeleteQuery(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("delete from sessions where expires_at < now")

	c.Assert(got, qt.DeepEquals, []string{
		"delete",
		"from sessions",
		"where expires_at < now",
	})
}

func TestFormatQueryForDisplayFormatsUnionAllAndOffset(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select id from archived_events union all select id from events order by id offset 100 limit 50")

	c.Assert(got, qt.DeepEquals, []string{
		"select id",
		"from archived_events",
		"union all select id",
		"from events",
		"order by id",
		"offset 100",
		"limit 50",
	})
}

func TestFormatQueryForDisplayMatchesClausesCaseInsensitively(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("SELECT id FROM invoices WHERE status = $1 AND archived = false ORDER BY id LIMIT 10")

	c.Assert(got, qt.DeepEquals, []string{
		"SELECT id",
		"FROM invoices",
		"WHERE status = $1",
		"  AND archived = false",
		"ORDER BY id",
		"LIMIT 10",
	})
}

func TestFormatQueryForDisplayDoesNotSplitAndInsideQuotedLiteral(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select 'research and development' from dual")

	c.Assert(got, qt.DeepEquals, []string{"select 'research and development' from dual"})
}

func TestFormatQueryForDisplayPreservesWhitespaceInsideQuotedLiteral(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select 'a  b' from t")

	c.Assert(got, qt.DeepEquals, []string{"select 'a  b' from t"})
}

func TestFormatQueryForDisplayDoesNotSplitFromInsideQuotedLiteral(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select * from notes where note = 'copied from admin'")

	c.Assert(got, qt.DeepEquals, []string{"select * from notes where note = 'copied from admin'"})
}

func TestFormatQueryForDisplayDoesNotSplitInsideDoubleQuotedText(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay(`select "research and development" from dual`)

	c.Assert(got, qt.DeepEquals, []string{`select "research and development" from dual`})
}

func TestFormatQueryForDisplayDoesNotSplitInsideBacktickQuotedText(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select `from` from t")

	c.Assert(got, qt.DeepEquals, []string{"select `from` from t"})
}

func TestFormatQueryForDisplayDoesNotSplitInsideDollarQuotedText(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select $$ where and $$ as text from t")

	c.Assert(got, qt.DeepEquals, []string{"select $$ where and $$ as text from t"})
}

func TestFormatQueryForDisplayDoesNotSplitInsideTaggedDollarQuotedText(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select $tag$ from hidden $tag$ from t")

	c.Assert(got, qt.DeepEquals, []string{"select $tag$ from hidden $tag$ from t"})
}

func TestFormatQueryForDisplayDoesNotSplitClausesInsideLineComment(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select 1 -- from hidden where id = 1")

	c.Assert(got, qt.DeepEquals, []string{"select 1 -- from hidden where id = 1"})
}

func TestFormatQueryForDisplayPreservesWhitespaceInsideLineComment(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select 1 --  from hidden")

	c.Assert(got, qt.DeepEquals, []string{"select 1 --  from hidden"})
}

func TestFormatQueryForDisplayDoesNotSplitClausesInsideHashComment(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select 1 # from hidden where id = 1")

	c.Assert(got, qt.DeepEquals, []string{"select 1 # from hidden where id = 1"})
}

func TestFormatQueryForDisplayDoesNotSplitClausesInsideBlockComment(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select 1 /* from hidden */ where id = 1")

	c.Assert(got, qt.DeepEquals, []string{"select 1 /* from hidden */ where id = 1"})
}

func TestFormatQueryForDisplayDoesNotSplitClausesInsideFunctionExpression(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select extract(day from created_at) from t")

	c.Assert(got, qt.DeepEquals, []string{"select extract(day from created_at) from t"})
}

func TestFormatQueryForDisplayDoesNotSplitClausesInsideWindowExpression(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select row_number() over (order by created_at) from t")

	c.Assert(got, qt.DeepEquals, []string{"select row_number() over (order by created_at) from t"})
}

func TestFormatQueryForDisplayDoesNotSplitBetweenCondition(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select * from t where created_at between $1 and $2")

	c.Assert(got, qt.DeepEquals, []string{"select * from t where created_at between $1 and $2"})
}

func TestFormatQueryForDisplayDoesNotSplitNotBetweenCondition(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select * from t where created_at not between $1 and $2")

	c.Assert(got, qt.DeepEquals, []string{"select * from t where created_at not between $1 and $2"})
}

func TestFormatQueryForDisplayFallsBackForCTE(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("with recent as (select * from t) select * from recent where id = $1")

	c.Assert(got, qt.DeepEquals, []string{"with recent as (select * from t) select * from recent where id = $1"})
}

func TestFormatQueryForDisplayFallsBackForMultilineInput(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select 1\nselect 2")

	c.Assert(got, qt.DeepEquals, []string{"select 1", "select 2"})
}

func TestFormatQueryForDisplayPreservesMultilineIndentation(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select 1\n  from t\nwhere a = 1")

	c.Assert(got, qt.DeepEquals, []string{"select 1", "  from t", "where a = 1"})
}

func TestFormatQueryForDisplayDropsSingleTerminalNewline(t *testing.T) {
	c := qt.New(t)
	got := formatQueryForDisplay("select 1\n  from t\n")

	c.Assert(got, qt.DeepEquals, []string{"select 1", "  from t"})
}

func TestQueryDisplayLinesPreservesAuthoredBlankLines(t *testing.T) {
	c := qt.New(t)
	got := queryDisplayLines("select 1\n\nselect 2", 80)

	c.Assert(got, qt.DeepEquals, []string{"select 1", "", "select 2"})
}

func TestQueryDisplayLinesDropsSingleTerminalNewline(t *testing.T) {
	c := qt.New(t)
	got := queryDisplayLines("select 1\n  from t\n", 80)

	c.Assert(got, qt.DeepEquals, []string{"select 1", "  from t"})
}
