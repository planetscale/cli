package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fatih/color"
	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func reportClient(engine ps.DatabaseEngine, metrics *mock.MetricsService) *ps.Client {
	return &ps.Client{
		Databases: &mock.DatabaseService{
			GetFn: func(context.Context, *ps.GetDatabaseRequest) (*ps.Database, error) {
				return &ps.Database{Name: "mydb", Kind: engine}, nil
			},
		},
		Metrics: metrics,
	}
}

func TestReportCmd_MySQLHumanUsesCuratedSections(t *testing.T) {
	c := qt.New(t)
	var requests []*ps.GetMetricSeriesRequest
	service := &mock.MetricsService{
		GetSeriesFn: func(_ context.Context, req *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error) {
			requests = append(requests, req)
			return sampleSeries(), nil
		},
	}

	var buf bytes.Buffer
	cmd := ReportCmd(metricsTestHelper(&buf, printer.Human, reportClient(ps.DatabaseEngineMySQL, service)))
	cmd.SetArgs([]string{"mydb", "main", "--period", "1d"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(requests, qt.HasLen, len(mysqlReportSections))
	c.Assert(service.GetInstantFnInvoked, qt.IsFalse)

	for i, definition := range mysqlReportSections {
		c.Assert(requests[i].Metrics, qt.DeepEquals, definition.Metrics)
		c.Assert(requests[i].Period, qt.Equals, "1d")
		c.Assert(buf.String(), qt.Contains, definition.Name)
	}
	c.Assert(buf.String(), qt.Contains, "Metrics report for planetscale/mydb/main")
	c.Assert(buf.String(), qt.Not(qt.Contains), "(MySQL)")
	c.Assert(buf.String(), qt.Not(qt.Contains), "##")
}

func TestReportCmd_PostgresJSONIncludesSeriesAndInstantSections(t *testing.T) {
	c := qt.New(t)
	var seriesRequests []*ps.GetMetricSeriesRequest
	var instantRequests []*ps.GetInstantMetricsRequest
	service := &mock.MetricsService{
		GetSeriesFn: func(_ context.Context, req *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error) {
			seriesRequests = append(seriesRequests, req)
			return sampleSeries(), nil
		},
		GetInstantFn: func(_ context.Context, req *ps.GetInstantMetricsRequest) (*ps.InstantMetrics, error) {
			instantRequests = append(instantRequests, req)
			return sampleInstantMetrics(), nil
		},
	}

	var buf bytes.Buffer
	cmd := ReportCmd(metricsTestHelper(&buf, printer.JSON, reportClient(ps.DatabaseEnginePostgres, service)))
	cmd.SetArgs([]string{"mydb", "main", "--period", "7d", "--steps", "96"})
	c.Assert(cmd.Execute(), qt.IsNil)

	wantSeries, wantInstant := 0, 0
	for _, definition := range postgresReportSections {
		switch definition.Kind {
		case reportSeriesSection:
			c.Assert(seriesRequests[wantSeries].Metrics, qt.DeepEquals, definition.Metrics)
			c.Assert(seriesRequests[wantSeries].Steps, qt.Equals, 96)
			wantSeries++
		case reportInstantSection:
			c.Assert(instantRequests[wantInstant].Metrics, qt.DeepEquals, definition.Metrics)
			wantInstant++
		}
	}
	c.Assert(seriesRequests, qt.HasLen, wantSeries)
	c.Assert(instantRequests, qt.HasLen, wantInstant)

	var report metricsReport
	c.Assert(json.Unmarshal(buf.Bytes(), &report), qt.IsNil)
	c.Assert(report.Type, qt.Equals, "MetricsReport")
	c.Assert(report.Organization, qt.Equals, "planetscale")
	c.Assert(report.Engine, qt.Equals, ps.DatabaseEnginePostgres)
	c.Assert(report.Period, qt.Equals, "7d")
	c.Assert(report.Steps, qt.Equals, 96)
	c.Assert(report.Sections, qt.HasLen, len(postgresReportSections))
	c.Assert(report.Sections[len(report.Sections)-1].Kind, qt.Equals, reportInstantSection)
}

func TestReportCmd_CustomRangeReplacesPeriod(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetSeriesFn: func(_ context.Context, req *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error) {
			c.Assert(req.Period, qt.Equals, "")
			c.Assert(req.From, qt.Equals, "2026-08-17T00:00:00Z")
			c.Assert(req.To, qt.Equals, "2026-08-18T00:00:00Z")
			return sampleSeries(), nil
		},
	}

	cmd := ReportCmd(metricsTestHelper(&bytes.Buffer{}, printer.JSON, reportClient(ps.DatabaseEngineMySQL, service)))
	cmd.SetArgs([]string{"mydb", "main", "--from", "2026-08-17T00:00:00Z", "--to", "2026-08-18T00:00:00Z"})
	c.Assert(cmd.Execute(), qt.IsNil)
}

func TestReportCmd_CSVIncludesSection(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{
		GetSeriesFn: func(context.Context, *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error) {
			return sampleSeries(), nil
		},
	}

	var buf bytes.Buffer
	cmd := ReportCmd(metricsTestHelper(&buf, printer.CSV, reportClient(ps.DatabaseEngineMySQL, service)))
	cmd.SetArgs([]string{"mydb", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	c.Assert(lines[0], qt.Equals, "section,kind,timestamp,metric,series,dimensions,value")
	c.Assert(lines[1], qt.Contains, "Workload, errors, and traffic control")
}

func TestReportCmd_NoColorProducesPlainSectionHeadings(t *testing.T) {
	c := qt.New(t)
	oldNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = oldNoColor })

	service := &mock.MetricsService{
		GetSeriesFn: func(context.Context, *ps.GetMetricSeriesRequest) (*ps.MetricSeries, error) {
			return sampleSeries(), nil
		},
	}
	var buf bytes.Buffer
	cmd := ReportCmd(metricsTestHelper(&buf, printer.Human, reportClient(ps.DatabaseEngineMySQL, service)))
	cmd.SetArgs([]string{"mydb", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Not(qt.Contains), "\x1b[")
	c.Assert(buf.String(), qt.Not(qt.Contains), "##")
	c.Assert(buf.String(), qt.Contains, "Latency and execution time\n")
}

func TestReportCmd_RejectsUnsupportedEngineBeforeFetchingMetrics(t *testing.T) {
	c := qt.New(t)
	service := &mock.MetricsService{}
	cmd := ReportCmd(metricsTestHelper(&bytes.Buffer{}, printer.JSON, reportClient(ps.DatabaseEngine("sqlite"), service)))
	cmd.SetArgs([]string{"mydb", "main"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `database engine "sqlite" is not supported by metrics report`)
	c.Assert(service.GetSeriesFnInvoked, qt.IsFalse)
	c.Assert(service.GetInstantFnInvoked, qt.IsFalse)
}
