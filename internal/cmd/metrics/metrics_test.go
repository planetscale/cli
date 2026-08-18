package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"

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
