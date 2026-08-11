package insights

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
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
			values := url.Values{}
			listOpts := &ps.ListOptions{URLValues: &values}
			for _, opt := range opts {
				c.Assert(opt(listOpts), qt.IsNil)
			}
			c.Assert(values.Get("from"), qt.Equals, "2026-07-24T07:00:00Z")
			c.Assert(values.Get("to"), qt.Equals, "2026-07-25T07:00:00Z")
			c.Assert(values.Get("period"), qt.Equals, "")
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
	cmd.SetArgs([]string{
		"mydb", "main",
		"--from", "2026-07-24T00:00:00-07:00",
		"--to", "2026-07-25T00:00:00-07:00",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListAnomaliesFnInvoked, qt.IsTrue)

	var out []map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out[0]["id"], qt.Equals, "a1")
	c.Assert(out[0]["active"], qt.Equals, true)
}

func TestInsights_AnomaliesCmd_Period(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		ListAnomaliesFn: func(ctx context.Context, req *ps.ListAnomaliesRequest, opts ...ps.ListOption) ([]*ps.Anomaly, error) {
			values := url.Values{}
			listOpts := &ps.ListOptions{URLValues: &values}
			for _, opt := range opts {
				c.Assert(opt(listOpts), qt.IsNil)
			}
			c.Assert(values.Get("period"), qt.Equals, "1d")
			c.Assert(values.Get("from"), qt.Equals, "")
			c.Assert(values.Get("to"), qt.Equals, "")
			return nil, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := AnomaliesCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "--period", "1d"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListAnomaliesFnInvoked, qt.IsTrue)
}

func TestInsights_AnomaliesCmd_GetByID(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		GetAnomalyFn: func(ctx context.Context, req *ps.GetAnomalyRequest) (*ps.Anomaly, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.AnomalyID, qt.Equals, "4888442")
			return &ps.Anomaly{
				ID:                 "4888442",
				PeriodStart:        time.Date(2026, 7, 24, 19, 15, 0, 0, time.UTC),
				PeriodEnd:          time.Date(2026, 7, 25, 0, 2, 0, 0, time.UTC),
				MinutesInViolation: 234,
				Correlations: []*ps.AnomalyCorrelation{{
					CorrelationCoefficient: 0.98,
					Keyspace:               "game",
					Fingerprint:            "abc123",
					NormalizedSQL:          "select * from spaceship where size > ?",
					TabletType:             "primary",
				}},
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})
	cmd := AnomaliesCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "4888442"})

	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetAnomalyFnInvoked, qt.IsTrue)
	var out map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out["id"], qt.Equals, "4888442")
	correlations := out["correlations"].([]any)
	c.Assert(correlations[0].(map[string]any)["fingerprint"], qt.Equals, "abc123")
}

func TestInsights_AnomaliesCmd_InvalidTimeSelection(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "to without from",
			args: []string{"mydb", "main", "--to", "2026-07-25T00:00:00Z"},
			want: "--to requires --from",
		},
		{
			name: "invalid from",
			args: []string{"mydb", "main", "--from", "July 24"},
			want: "invalid --from",
		},
		{
			name: "invalid to",
			args: []string{"mydb", "main", "--from", "2026-07-24T00:00:00Z", "--to", "July 25"},
			want: "invalid --to",
		},
		{
			name: "reversed range",
			args: []string{"mydb", "main", "--from", "2026-07-25T00:00:00Z", "--to", "2026-07-24T00:00:00Z"},
			want: "--from must be before --to",
		},
		{
			name: "unknown period",
			args: []string{"mydb", "main", "--period", "24h"},
			want: `invalid --period "24h"`,
		},
		{
			name: "period and range",
			args: []string{"mydb", "main", "--period", "1d", "--from", "2026-07-24T00:00:00Z"},
			want: "if any flags in the group",
		},
		{
			name: "detail with range",
			args: []string{"mydb", "main", "4888442", "--period", "1d"},
			want: "cannot be used when retrieving an anomaly by ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			var buf bytes.Buffer
			ch := testHelper(&buf, printer.JSON, &ps.Client{})

			cmd := AnomaliesCmd(ch)
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			c.Assert(err, qt.IsNotNil)
			c.Assert(err.Error(), qt.Contains, tt.want)
		})
	}
}

func TestAnomalyTimeRange_DateOnly(t *testing.T) {
	c := qt.New(t)
	location := time.FixedZone("PDT", -7*60*60)
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, location)

	from, to, err := anomalyTimeRange("07/23", "07/25", now)

	c.Assert(err, qt.IsNil)
	c.Assert(from, qt.DeepEquals, time.Date(2026, 7, 23, 0, 0, 0, 0, location))
	c.Assert(to, qt.DeepEquals, time.Date(2026, 7, 26, 0, 0, 0, 0, location))
}

func TestAnomalyTimeRange_HumanReadableFormats(t *testing.T) {
	c := qt.New(t)
	location := time.FixedZone("PDT", -7*60*60)
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, location)
	tests := []struct {
		value string
		want  time.Time
	}{
		{value: "2026-07-23", want: time.Date(2026, 7, 23, 0, 0, 0, 0, location)},
		{value: "7/23/2026", want: time.Date(2026, 7, 23, 0, 0, 0, 0, location)},
		{value: "7/23", want: time.Date(2026, 7, 23, 0, 0, 0, 0, location)},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			parsed, dateOnly, err := parseAnomalyTime(tt.value, now)

			c.Assert(err, qt.IsNil)
			c.Assert(dateOnly, qt.IsTrue)
			c.Assert(parsed, qt.DeepEquals, tt.want)
		})
	}
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
