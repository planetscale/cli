package insights

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func testHelper(buf *bytes.Buffer, format printer.Format, client *ps.Client) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)

	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return client, nil
		},
	}
}

func TestInsights_QueriesCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		ListQueriesFn: func(ctx context.Context, req *ps.ListQueryInsightsRequest, opts ...ps.ListOption) ([]*ps.QueryInsight, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.Branch, qt.Equals, "main")
			return []*ps.QueryInsight{{
				ID:                     "q1",
				NormalizedSQL:          "select * from users where id = ?",
				QueryCount:             10,
				SumTotalDurationMillis: 123.456,
				P99Latency:             4.2,
				LastRunAt:              time.Date(2026, 7, 22, 23, 22, 0, 0, time.UTC),
			}}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := QueriesCmd(ch)
	cmd.SetArgs([]string{"mydb", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListQueriesFnInvoked, qt.IsTrue)

	var out []map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out, qt.HasLen, 1)
	c.Assert(out[0]["normalized_sql"], qt.Equals, "select * from users where id = ?")
}

func TestInsights_QueriesCmd_InvalidSort(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{})

	cmd := QueriesCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "--sort", "bogus"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, `invalid --sort "bogus"`)
}

func TestInsights_ErrorsCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		ListErrorsFn: func(ctx context.Context, req *ps.ListQueryInsightsErrorsRequest, opts ...ps.ListOption) ([]*ps.QueryInsightError, error) {
			return []*ps.QueryInsightError{{
				ID:           "e1",
				ErrorCount:   4,
				ErrorMessage: "relation \"widgets\" does not exist",
				StartedAt:    time.Date(2026, 7, 22, 14, 5, 0, 0, time.UTC),
			}}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.CSV, &ps.Client{QueryInsights: svc})

	cmd := ErrorsCmd(ch)
	cmd.SetArgs([]string{"mydb", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListErrorsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "relation")
}

func TestInsights_AnomaliesCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		ListAnomaliesFn: func(ctx context.Context, req *ps.ListAnomaliesRequest, opts ...ps.ListOption) ([]*ps.Anomaly, error) {
			return []*ps.Anomaly{{
				ID:                 "a1",
				Active:             true,
				MinutesInViolation: 12,
				PeriodStart:        time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC),
			}}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := AnomaliesCmd(ch)
	cmd.SetArgs([]string{"mydb", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListAnomaliesFnInvoked, qt.IsTrue)

	var out []map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out[0]["id"], qt.Equals, "a1")
	c.Assert(out[0]["active"], qt.Equals, true)
}

func TestInsights_RecommendationsCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.SchemaRecommendationService{
		ListFn: func(ctx context.Context, req *ps.ListSchemaRecommendationsRequest, opts ...ps.ListOption) ([]*ps.SchemaRecommendation, error) {
			c.Assert(req.Database, qt.Equals, "mydb")
			return []*ps.SchemaRecommendation{{
				Number:             1,
				State:              "open",
				RecommendationType: "duplicate_index",
				Table:              "users",
				Title:              "Drop duplicate index idx_email",
				DDLStatement:       "ALTER TABLE users DROP INDEX idx_email",
			}}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{SchemaRecommendations: svc})

	cmd := RecommendationsCmd(ch)
	cmd.SetArgs([]string{"mydb"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)

	var out []map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out[0]["recommendation_type"], qt.Equals, "duplicate_index")
	c.Assert(out[0]["ddl_statement"], qt.Equals, "ALTER TABLE users DROP INDEX idx_email")
}

func TestInsights_RecommendationDismissCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.SchemaRecommendationService{
		DismissFn: func(ctx context.Context, req *ps.DismissSchemaRecommendationRequest) (*ps.SchemaRecommendation, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.ID, qt.Equals, "42")
			c.Assert(req.Reason, qt.Equals, "false positive")
			return &ps.SchemaRecommendation{
				ID:                 "rec-42",
				Number:             42,
				State:              "dismissed",
				RecommendationType: "unused_index",
				Table:              "users",
				Title:              "Drop unused index",
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{SchemaRecommendations: svc})

	cmd := RecommendationDismissCmd(ch)
	cmd.SetArgs([]string{"mydb", "42", "--force", "--reason", "false positive"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DismissFnInvoked, qt.IsTrue)

	var out map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out["state"], qt.Equals, "dismissed")
	c.Assert(out["number"], qt.Equals, float64(42))
}

func TestInsights_RecommendationDismissCmd_RequiresForceInJSON(t *testing.T) {
	c := qt.New(t)

	svc := &mock.SchemaRecommendationService{}
	ch := testHelper(&bytes.Buffer{}, printer.JSON, &ps.Client{SchemaRecommendations: svc})

	cmd := RecommendationDismissCmd(ch)
	cmd.SetArgs([]string{"mydb", "42"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*run with --force.*`)
	c.Assert(svc.DismissFnInvoked, qt.IsFalse)
}

func TestInsights_QuerySamplesCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		ListQuerySamplesFn: func(ctx context.Context, req *ps.ListQuerySamplesRequest, opts ...ps.ListOption) ([]*ps.QuerySample, error) {
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.Fingerprint, qt.Equals, "b129e8fa")
			return []*ps.QuerySample{{
				ID:                  "exec-1",
				Fingerprint:         "b129e8fa",
				NormalizedSQL:       "select 1",
				Username:            "app",
				TotalDurationMillis: 1.5,
				StartedAt:           time.Date(2026, 8, 11, 18, 0, 0, 0, time.UTC),
				Tags:                []ps.QuerySampleTag{{Name: "Sapp", Value: "web"}},
			}}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := QuerySamplesCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "b129e8fa", "--keyspace", "mydb"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListQuerySamplesFnInvoked, qt.IsTrue)

	var out []map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out[0]["id"], qt.Equals, "exec-1")
	c.Assert(out[0]["username"], qt.Equals, "app")
}


func TestInsights_QuerySamplesCmd_RequiresKeyspace(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{}
	ch := testHelper(&bytes.Buffer{}, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := QuerySamplesCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "b129e8fa"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "keyspace")
	c.Assert(svc.ListQuerySamplesFnInvoked, qt.IsFalse)
}

func TestInsights_TagsCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		ListTagsFn: func(ctx context.Context, req *ps.ListQueryTagsRequest, opts ...ps.ListOption) ([]*ps.QueryTag, error) {
			return []*ps.QueryTag{{
				ID:         "Sapp",
				Name:       "app",
				Source:     "sql",
				QueryCount: 100,
				Values:     []ps.QueryTagValue{{Name: "web", QueryCount: 80, Kind: "literal"}},
			}}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := TagsCmd(ch)
	cmd.SetArgs([]string{"mydb", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListTagsFnInvoked, qt.IsTrue)

	var out []map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out[0]["name"], qt.Equals, "app")
	c.Assert(out[0]["id"], qt.Equals, "Sapp")
}

func TestInsights_TagSummariesCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		ListTagsFn: func(ctx context.Context, req *ps.ListQueryTagsRequest, opts ...ps.ListOption) ([]*ps.QueryTag, error) {
			return []*ps.QueryTag{
				{ID: "Busername", Name: "username", Source: "system"},
				{ID: "Sapp", Name: "app", Source: "sql"},
			}, nil
		},
		ListTagSummariesFn: func(ctx context.Context, req *ps.ListTagSummariesRequest, opts ...ps.ListOption) ([]*ps.TagSummary, error) {
			c.Assert(req.Tags, qt.DeepEquals, []string{"Busername", "Sapp"})
			return []*ps.TagSummary{{
				Dimensions:             map[string]string{"Busername": "alice", "Sapp": "web"},
				QueryCount:             10,
				SumTotalDurationMillis: 42,
			}}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := TagSummariesCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "--tags", "username", "--tags", "app"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListTagsFnInvoked, qt.IsTrue)
	c.Assert(svc.ListTagSummariesFnInvoked, qt.IsTrue)

	var out []map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out[0]["query_count"], qt.Equals, float64(10))
}

func TestInsights_TagShowCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		ListTagsFn: func(ctx context.Context, req *ps.ListQueryTagsRequest, opts ...ps.ListOption) ([]*ps.QueryTag, error) {
			return []*ps.QueryTag{{ID: "Sapp", Name: "app", Source: "sql"}}, nil
		},
		GetTagFn: func(ctx context.Context, req *ps.GetQueryTagRequest, opts ...ps.ListOption) (*ps.QueryTag, error) {
			c.Assert(req.Tag, qt.Equals, "Sapp")
			return &ps.QueryTag{
				ID:         "Sapp",
				Name:       "app",
				Source:     "sql",
				QueryCount: 100,
				Values:     []ps.QueryTagValue{{Name: "web", QueryCount: 80, Kind: "literal"}},
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := TagShowCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "app"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetTagFnInvoked, qt.IsTrue)

	var out map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out["id"], qt.Equals, "Sapp")
}
