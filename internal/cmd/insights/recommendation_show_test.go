package insights

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func TestInsights_RecommendationShowCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.SchemaRecommendationService{
		GetFn: func(ctx context.Context, req *ps.GetSchemaRecommendationRequest) (*ps.SchemaRecommendation, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.ID, qt.Equals, "42")
			return &ps.SchemaRecommendation{
				ID:                 "rec-42",
				Number:             42,
				State:              "open",
				RecommendationType: "unused_index",
				Table:              "users",
				Keyspace:           "main",
				Title:              "Drop unused index idx_email",
				DDLStatement:       "ALTER TABLE users DROP INDEX idx_email",
				HtmlURL:            "https://app.planetscale.com/planetscale/mydb/insights/recommendations/42",
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{SchemaRecommendations: svc})

	cmd := RecommendationShowCmd(ch)
	cmd.SetArgs([]string{"mydb", "42"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)

	var out map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out["number"], qt.Equals, float64(42))
	c.Assert(out["ddl_statement"], qt.Equals, "ALTER TABLE users DROP INDEX idx_email")
	c.Assert(out["title"], qt.Equals, "Drop unused index idx_email")
}

func TestInsights_RecommendationShowCmd_HumanIncludesFullDDL(t *testing.T) {
	c := qt.New(t)

	ddl := "ALTER TABLE users DROP INDEX idx_email, DROP INDEX idx_email_dup"
	svc := &mock.SchemaRecommendationService{
		GetFn: func(ctx context.Context, req *ps.GetSchemaRecommendationRequest) (*ps.SchemaRecommendation, error) {
			c.Assert(req.ID, qt.Equals, "1")
			return &ps.SchemaRecommendation{
				Number:             1,
				State:              "open",
				RecommendationType: "duplicate_index",
				Table:              "users",
				Keyspace:           "main",
				Title:              "Drop duplicate index",
				DDLStatement:       ddl,
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.Human, &ps.Client{SchemaRecommendations: svc})
	ch.Printer.SetHumanOutput(&buf)

	cmd := RecommendationShowCmd(ch)
	cmd.SetArgs([]string{"mydb", "1"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, ddl)
	c.Assert(buf.String(), qt.Contains, "duplicate_index")
}

func TestInsights_RecommendationShowCmd_NotFound(t *testing.T) {
	c := qt.New(t)

	svc := &mock.SchemaRecommendationService{
		GetFn: func(ctx context.Context, req *ps.GetSchemaRecommendationRequest) (*ps.SchemaRecommendation, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{SchemaRecommendations: svc})

	cmd := RecommendationShowCmd(ch)
	cmd.SetArgs([]string{"mydb", "99"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
	c.Assert(err.Error(), qt.Contains, "99")
}

func TestInsights_RecommendationsCmd_HasShowSubcommand(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{})

	cmd, _, err := RecommendationsCmd(ch).Find([]string{"show"})
	c.Assert(err, qt.IsNil)
	c.Assert(cmd.Name(), qt.Equals, "show")
}
