package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func metricsTestHelper(buf *bytes.Buffer, format printer.Format, client *ps.Client) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	if format == printer.Human {
		p.SetHumanOutput(buf)
	}
	p.SetResourceOutput(buf)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return client, nil
		},
	}
}

func sampleSeries() *ps.MetricSeries {
	return &ps.MetricSeries{
		Type:      "MetricSeries",
		StartDate: time.Date(2026, 8, 18, 16, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 8, 18, 17, 0, 0, 0, time.UTC),
		Interval:  60,
		Series: []*ps.TimeSeries{{
			Type:   "TimeSeries",
			Metric: "queries",
			Label:  "Queries",
			Labels: map[string]string{},
			Points: [][]float64{{1787068800, 912}, {1787068860, 1284}, {1787068920, 1903}},
		}},
	}
}

func TestShowCmd_JSONPreservesSeries(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetSeriesFn: func(ctx context.Context, req *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.Metrics, qt.DeepEquals, []string{"queries", "latency_p99"})
			c.Assert(req.Period, qt.Equals, "1h")
			c.Assert(req.Role, qt.Equals, "primary")
			return sampleSeries(), nil
		},
	}

	var buf bytes.Buffer
	cmd := ShowCmd(metricsTestHelper(&buf, printer.JSON, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{"mydb", "main", "--metric", "queries,latency_p99", "--period", "1h", "--role", "primary"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.GetSeriesFnInvoked, qt.IsTrue)

	var response ps.MetricSeries
	c.Assert(json.Unmarshal(buf.Bytes(), &response), qt.IsNil)
	c.Assert(response.Type, qt.Equals, "MetricSeries")
	c.Assert(response.Series[0].Points, qt.HasLen, 3)
}

func TestShowCmd_ForwardsOpaqueQueryID(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetSeriesFn: func(ctx context.Context, req *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error) {
			c.Assert(req.QueryIDs, qt.DeepEquals, []string{"59801dae501c"})
			return sampleSeries(), nil
		},
	}

	var buf bytes.Buffer
	cmd := ShowCmd(metricsTestHelper(&buf, printer.JSON, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{"mydb", "main", "--metric", "queries", "--query-id", "59801dae501c"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.GetSeriesFnInvoked, qt.IsTrue)
}

func TestShowCmd_HumanSummarizesSeries(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetSeriesFn: func(context.Context, *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error) {
			return sampleSeries(), nil
		},
	}

	var buf bytes.Buffer
	cmd := ShowCmd(metricsTestHelper(&buf, printer.Human, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{"mydb", "main", "--metric", "queries", "--period", "1h"})
	c.Assert(cmd.Execute(), qt.IsNil)

	output := buf.String()
	c.Assert(output, qt.Contains, "Metrics for mydb/main")
	c.Assert(output, qt.Contains, "60s")
	c.Assert(output, qt.Contains, "Queries")
	c.Assert(output, qt.Contains, "1,903")
	c.Assert(output, qt.Contains, "▁")
}

func TestShowCmd_CSVFlattensPoints(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetSeriesFn: func(context.Context, *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error) {
			return sampleSeries(), nil
		},
	}

	var buf bytes.Buffer
	cmd := ShowCmd(metricsTestHelper(&buf, printer.CSV, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{"mydb", "main", "--metric", "queries"})
	c.Assert(cmd.Execute(), qt.IsNil)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	c.Assert(lines, qt.HasLen, 4)
	c.Assert(lines[0], qt.Equals, "timestamp,metric,label,labels,value")
	c.Assert(lines[1], qt.Contains, "2026-08-18T16:00:00Z,queries,Queries,{}")
}

func TestShowCmd_RejectsIncompleteCustomRange(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{}
	cmd := ShowCmd(metricsTestHelper(&bytes.Buffer{}, printer.JSON, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{"mydb", "main", "--metric", "queries", "--from", "2026-08-18T16:00:00Z"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, ".*--from and --to must be used together.*")
	c.Assert(service.GetSeriesFnInvoked, qt.IsFalse)
}

func TestHumanMetricFormatting(t *testing.T) {
	c := qt.New(t)
	tests := []struct {
		metric string
		value  float64
		want   string
	}{
		{metric: "queries", value: 1284, want: "1,284"},
		{metric: "latency_p99", value: 18.2, want: "18.2 ms"},
		{metric: "planetscale_volume_usage_percentage", value: 71.4, want: "71.4%"},
		{metric: "planetscale_primary_storage_usage", value: 1073741824, want: "1.0 GiB"},
		{metric: "planetscale_edge_bytes_received", value: 10485760, want: "10 MiB"},
		{metric: "planetscale_edge_bytes_received_rate", value: 10485760, want: "10 MiB/s"},
		{metric: "planetscale_edge_bytes_sent_rate", value: 1536, want: "1.5 KiB/s"},
		{metric: "block_cache_hit_ratio", value: 99.2, want: "99.2%"},
		{metric: "vtgate_cpu_by_az", value: 42.3, want: "42.3%"},
	}

	for _, test := range tests {
		c.Run(test.metric, func(c *qt.C) {
			c.Assert(formatMetricValue(test.metric, test.value), qt.Equals, test.want)
		})
	}
	c.Assert(humanMetricName("latency_p99"), qt.Equals, "Latency p99")
	c.Assert(formatNumber(123.456789), qt.Equals, "123.5")
	c.Assert(formatNumber(12.3456789), qt.Equals, "12.35")
	c.Assert(formatNumber(0.00123456789), qt.Equals, "0.001235")
}

func TestSeriesSummaryRowsFormatsByteRates(t *testing.T) {
	c := qt.New(t)
	response := sampleSeries()
	response.Series[0].Metric = "planetscale_edge_bytes_received_rate"
	response.Series[0].Points = [][]float64{{1787068800, 10485760}}

	rows := seriesSummaryRows(response)
	c.Assert(rows, qt.HasLen, 1)
	c.Assert(rows[0].Latest, qt.Equals, "10 MiB/s")
	c.Assert(rows[0].Min, qt.Equals, "10 MiB/s")
	c.Assert(rows[0].Avg, qt.Equals, "10 MiB/s")
	c.Assert(rows[0].Max, qt.Equals, "10 MiB/s")
}

func sampleInstantMetrics() *ps.InstantMetrics {
	return &ps.InstantMetrics{
		Type:   "InstantMetrics",
		Branch: map[string]any{"id": "branch-id", "name": "main"},
		Metrics: []*ps.InstantMetric{{
			Metric: "planetscale_volume_usage_percentage",
			Label:  "volume_usage",
			Values: []map[string]any{{"pod": "postgres-0", "role": "primary", "value": 71.4}},
		}},
	}
}

func TestInstantCmd_HumanFormatsValues(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetInstantFn: func(ctx context.Context, req *ps.GetInstantMetricsRequest) (*ps.InstantMetrics, error) {
			c.Assert(req.Role, qt.Equals, "primary")
			return sampleInstantMetrics(), nil
		},
	}

	var buf bytes.Buffer
	cmd := InstantCmd(metricsTestHelper(&buf, printer.Human, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{"mydb", "main", "--metric", "planetscale_volume_usage_percentage", "--role", "primary"})
	c.Assert(cmd.Execute(), qt.IsNil)

	output := buf.String()
	c.Assert(output, qt.Contains, "Volume usage")
	c.Assert(output, qt.Contains, "pod=postgres-0, role=primary")
	c.Assert(output, qt.Contains, "71.4%")
}

func TestInstantCmd_JSONPreservesEnvelope(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetInstantFn: func(context.Context, *ps.GetInstantMetricsRequest) (*ps.InstantMetrics, error) {
			return sampleInstantMetrics(), nil
		},
	}

	var buf bytes.Buffer
	cmd := InstantCmd(metricsTestHelper(&buf, printer.JSON, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{"mydb", "main", "--metric", "planetscale_volume_usage_percentage"})
	c.Assert(cmd.Execute(), qt.IsNil)

	var response ps.InstantMetrics
	c.Assert(json.Unmarshal(buf.Bytes(), &response), qt.IsNil)
	c.Assert(response.Type, qt.Equals, "InstantMetrics")
	c.Assert(response.Metrics[0].Values[0]["value"], qt.Equals, 71.4)
}

func TestInstantCmd_CSVFlattensValues(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetInstantFn: func(context.Context, *ps.GetInstantMetricsRequest) (*ps.InstantMetrics, error) {
			return sampleInstantMetrics(), nil
		},
	}

	var buf bytes.Buffer
	cmd := InstantCmd(metricsTestHelper(&buf, printer.CSV, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{"mydb", "main", "--metric", "planetscale_volume_usage_percentage"})
	c.Assert(cmd.Execute(), qt.IsNil)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	c.Assert(lines, qt.HasLen, 2)
	c.Assert(lines[0], qt.Equals, "metric,label,dimensions,value")
	c.Assert(lines[1], qt.Contains, "planetscale_volume_usage_percentage,volume_usage")
	c.Assert(lines[1], qt.Contains, "71.4")
}

func TestQueriesCmd_ForwardsSupportedFilters(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetQuerySeriesFn: func(ctx context.Context, req *ps.GetQueryMetricSeriesRequest) (*ps.MetricSeries, error) {
			c.Assert(req.Metrics, qt.DeepEquals, []string{"queries", "latency_p99", "traffic_control_warnings"})
			c.Assert(req.QueryIDs, qt.HasLen, 0)
			c.Assert(req.Fingerprint, qt.Equals, "fingerprint-1")
			c.Assert(req.Keyspace, qt.Equals, "commerce")
			c.Assert(req.Period, qt.Equals, "1h")
			c.Assert(req.TabletType, qt.Equals, "replica")
			c.Assert(req.BudgetID, qt.Equals, "budget-1")
			c.Assert(req.RuleID, qt.Equals, "rule-1")
			c.Assert(req.Search, qt.Equals, "checkout")
			return sampleSeries(), nil
		},
	}

	var buf bytes.Buffer
	cmd := QueriesCmd(metricsTestHelper(&buf, printer.JSON, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{
		"mydb", "main",
		"--metric", "queries,latency_p99,traffic_control_warnings",
		"--fingerprint", "fingerprint-1",
		"--keyspace", "commerce",
		"--period", "1h",
		"--tablet-type", "replica",
		"--budget-id", "budget-1",
		"--rule-id", "rule-1",
		"--search", "checkout",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.GetQuerySeriesFnInvoked, qt.IsTrue)

	var response ps.MetricSeries
	c.Assert(json.Unmarshal(buf.Bytes(), &response), qt.IsNil)
	c.Assert(response.Series, qt.HasLen, 1)
}

func TestStorageMetricsCommands_PreserveResponse(t *testing.T) {
	c := qt.New(t)
	tests := []struct {
		name    string
		command func(*cmdutil.Helper) *cobra.Command
		service *mock.MetricsService
	}{
		{
			name:    "tables",
			command: TablesCmd,
			service: &mock.MetricsService{
				GetTablesFn: func(context.Context, *ps.GetBranchMetricsRequest) (json.RawMessage, error) {
					return json.RawMessage(`{"users":{"bytes":1048576}}`), nil
				},
			},
		},
		{
			name:    "keyspace tables",
			command: KeyspaceTablesCmd,
			service: &mock.MetricsService{
				GetKeyspaceTablesFn: func(context.Context, *ps.GetBranchMetricsRequest) (json.RawMessage, error) {
					return json.RawMessage(`{"commerce":{"users":{"bytes":1048576}}}`), nil
				},
			},
		},
	}

	for _, test := range tests {
		c.Run(test.name, func(c *qt.C) {
			var buf bytes.Buffer
			cmd := test.command(metricsTestHelper(&buf, printer.JSON, &ps.Client{Metrics: test.service}))
			cmd.SetArgs([]string{"mydb", "main"})
			c.Assert(cmd.Execute(), qt.IsNil)
			c.Assert(json.Valid(buf.Bytes()), qt.IsTrue)
			c.Assert(buf.String(), qt.Contains, "1048576")
		})
	}
}

func TestTabletsCmd_ForwardsSupportedFilters(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetTabletSeriesFn: func(ctx context.Context, req *ps.GetTabletMetricSeriesRequest) (*ps.MetricSeries, error) {
			c.Assert(req.Metrics, qt.DeepEquals, []string{"replication_lag", "pod_cpu_usage", "vreplication_lag"})
			c.Assert(req.From, qt.Equals, "2026-08-18T16:00:00Z")
			c.Assert(req.To, qt.Equals, "2026-08-18T17:00:00Z")
			c.Assert(req.Steps, qt.Equals, 60)
			c.Assert(req.Keyspace, qt.Equals, "commerce")
			c.Assert(req.Shard, qt.Equals, "-80")
			c.Assert(req.Pod, qt.Equals, "zone-a-0")
			c.Assert(req.Workflow, qt.Equals, "move-tables")
			return sampleSeries(), nil
		},
	}

	var buf bytes.Buffer
	cmd := TabletsCmd(metricsTestHelper(&buf, printer.JSON, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{
		"mydb", "main",
		"--metric", "replication_lag,pod_cpu_usage,vreplication_lag",
		"--from", "2026-08-18T16:00:00Z",
		"--to", "2026-08-18T17:00:00Z",
		"--steps", "60",
		"--keyspace", "commerce",
		"--shard", "-80",
		"--pod", "zone-a-0",
		"--workflow", "move-tables",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.GetTabletSeriesFnInvoked, qt.IsTrue)
}

func TestTabletsInstantCmd_UsesNestedUX(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetInstantTabletsFn: func(ctx context.Context, req *ps.GetInstantTabletMetricsRequest) (*ps.InstantMetrics, error) {
			c.Assert(req.Metrics, qt.DeepEquals, []string{"replication_lag", "primary_cpu_usage"})
			c.Assert(req.Keyspace, qt.Equals, "commerce")
			c.Assert(req.Shard, qt.Equals, "-80")
			return sampleInstantMetrics(), nil
		},
	}

	var buf bytes.Buffer
	cmd := TabletsCmd(metricsTestHelper(&buf, printer.JSON, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{
		"instant", "mydb", "main",
		"--metric", "replication_lag,primary_cpu_usage",
		"--keyspace", "commerce",
		"--shard", "-80",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.GetInstantTabletsFnInvoked, qt.IsTrue)
}

func TestTabletsCmd_DoesNotPrevalidateWorkflow(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetTabletSeriesFn: func(_ context.Context, req *ps.GetTabletMetricSeriesRequest) (*ps.MetricSeries, error) {
			c.Assert(req.Workflow, qt.Equals, "workflow-on-a-later-page")
			return sampleSeries(), nil
		},
	}

	cmd := TabletsCmd(metricsTestHelper(&bytes.Buffer{}, printer.JSON, &ps.Client{
		Metrics: service,
	}))
	cmd.SetArgs([]string{
		"mydb", "main",
		"--metric", "vreplication_lag",
		"--workflow", "workflow-on-a-later-page",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.GetTabletSeriesFnInvoked, qt.IsTrue)
}

func TestTagsCmd_ForwardsSupportedFilters(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetTagSeriesFn: func(ctx context.Context, req *ps.GetTagMetricSeriesRequest) (*ps.MetricSeries, error) {
			c.Assert(req.Metrics, qt.DeepEquals, []string{"queries", "latency_p99", "traffic_control_warnings"})
			c.Assert(req.TagSets, qt.DeepEquals, []map[string]string{
				{"Busername": "alice", "Senv": "production"},
				{"Busername": "bob"},
			})
			c.Assert(req.Period, qt.Equals, "1d")
			c.Assert(req.TabletType, qt.Equals, "primary")
			c.Assert(req.BudgetID, qt.Equals, "budget-1")
			c.Assert(req.RuleID, qt.Equals, "rule-1")
			c.Assert(req.Search, qt.Equals, "checkout")
			return sampleSeries(), nil
		},
	}

	var buf bytes.Buffer
	cmd := TagsCmd(metricsTestHelper(&buf, printer.JSON, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{
		"mydb", "main",
		"--metric", "queries,latency_p99,traffic_control_warnings",
		"--tag-set", "Busername=alice,Senv=production",
		"--tag-set", "Busername=bob",
		"--period", "1d",
		"--tablet-type", "primary",
		"--budget-id", "budget-1",
		"--rule-id", "rule-1",
		"--search", "checkout",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.GetTagSeriesFnInvoked, qt.IsTrue)
}

func TestQueriesCmd_ForwardsOpaqueQueryID(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetQuerySeriesFn: func(ctx context.Context, req *ps.GetQueryMetricSeriesRequest) (*ps.MetricSeries, error) {
			c.Assert(req.QueryIDs, qt.DeepEquals, []string{"59801dae501c"})
			return sampleSeries(), nil
		},
	}

	var buf bytes.Buffer
	cmd := QueriesCmd(metricsTestHelper(&buf, printer.JSON, &ps.Client{Metrics: service}))
	cmd.SetArgs([]string{"mydb", "main", "--metric", "queries", "--query-id", "59801dae501c"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(service.GetQuerySeriesFnInvoked, qt.IsTrue)
}

func TestParseTagSets(t *testing.T) {
	c := qt.New(t)

	sets, err := parseTagSets([]string{"Busername=alice,Senv=production", "Busername=bob"})
	c.Assert(err, qt.IsNil)
	c.Assert(sets, qt.DeepEquals, []map[string]string{
		{"Busername": "alice", "Senv": "production"},
		{"Busername": "bob"},
	})

	_, err = parseTagSets([]string{"alice"})
	c.Assert(err, qt.ErrorMatches, `invalid --tag-set "alice"; use key=value pairs with an Insights type prefix, for example Busername=alice`)

	_, err = parseTagSets([]string{"username=alice"})
	c.Assert(err, qt.ErrorMatches, `invalid tag key "username"; keys must start with B \(built-in\) or S \(custom\), for example Busername or Sapplication`)
}

func TestValidateQuerySelector(t *testing.T) {
	c := qt.New(t)
	validID := "59801dae501c"

	c.Assert(validateQuerySelector([]string{validID}, "", ""), qt.IsNil)
	c.Assert(validateQuerySelector(nil, "fingerprint", "commerce"), qt.IsNil)
	c.Assert(validateQuerySelector(nil, "", ""), qt.ErrorMatches, "select at least one query with --query-id or with --fingerprint and --keyspace")
	c.Assert(validateQuerySelector(nil, "fingerprint", ""), qt.ErrorMatches, "--fingerprint and --keyspace must be used together")
	c.Assert(validateQuerySelector([]string{validID}, "fingerprint", "commerce"), qt.ErrorMatches, "--query-id cannot be combined with --fingerprint or --keyspace")
}

func TestValidateTrafficControlMetricFilters(t *testing.T) {
	c := qt.New(t)

	c.Assert(validateTrafficControlMetricFilters([]string{"queries"}, "", ""), qt.IsNil)
	c.Assert(validateTrafficControlMetricFilters([]string{"traffic_control_warnings"}, "budget", ""), qt.IsNil)
	c.Assert(validateTrafficControlMetricFilters([]string{"traffic_control_throttled"}, "", "rule"), qt.IsNil)
	c.Assert(
		validateTrafficControlMetricFilters([]string{"queries"}, "budget", ""),
		qt.ErrorMatches,
		"--budget-id and --rule-id only apply to --metric traffic_control_warnings or traffic_control_throttled",
	)
}
