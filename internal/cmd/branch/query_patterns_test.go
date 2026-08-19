package branch

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

const queryPatternsCSV = "normalized_sql,query_count\nselect ?,10\n"

func queryPatternsTestHelper(org, baseURL string, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)
	p.SetHumanOutput(bytes.NewBuffer(nil))

	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{AccessToken: "token", Organization: org, BaseURL: baseURL},
		Client: func() (*ps.Client, error) {
			return ps.NewClient(ps.WithBaseURL(baseURL), ps.WithAccessToken("token"))
		},
	}
}

func queryPatternsServer(t *testing.T, c *qt.C, showStates []string) *httptest.Server {
	t.Helper()

	base := "/v1/organizations/my-org/databases/my-db/branches/my-branch/query-patterns"
	shows := 0
	mux := http.NewServeMux()

	mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.Header.Get("Authorization"), qt.Equals, "Bearer token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "report1", "state": "pending"})
	})

	mux.HandleFunc(base+"/report1", func(w http.ResponseWriter, r *http.Request) {
		state := showStates[min(shows, len(showStates)-1)]
		shows++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "report1", "state": state})
	})

	mux.HandleFunc(base+"/report1/download", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/blob", http.StatusFound)
	})

	mux.HandleFunc("/blob", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		gz := gzip.NewWriter(w)
		_, _ = gz.Write([]byte(queryPatternsCSV))
		_ = gz.Close()
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestBranch_QueryPatternsDownloadCmd(t *testing.T) {
	c := qt.New(t)

	prevInterval := queryPatternsPollInterval
	queryPatternsPollInterval = 10 * time.Millisecond
	t.Cleanup(func() { queryPatternsPollInterval = prevInterval })

	server := queryPatternsServer(t, c, []string{"pending", "completed"})

	var buf bytes.Buffer
	ch := queryPatternsTestHelper("my-org", server.URL, printer.JSON, &buf)

	output := filepath.Join(t.TempDir(), "report.csv")

	cmd := QueryPatternsCmd(ch)
	cmd.SetArgs([]string{"download", "my-db", "my-branch", "--output", output})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)

	data, err := os.ReadFile(output)
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, queryPatternsCSV)

	c.Assert(buf.String(), qt.JSONEquals, &QueryPatternsDownload{
		ID:    "report1",
		State: "completed",
		File:  output,
	})
}

func TestBranch_QueryPatternsDownloadCmd_Stdout(t *testing.T) {
	c := qt.New(t)

	prevInterval := queryPatternsPollInterval
	queryPatternsPollInterval = 10 * time.Millisecond
	t.Cleanup(func() { queryPatternsPollInterval = prevInterval })

	server := queryPatternsServer(t, c, []string{"pending", "completed"})

	var buf bytes.Buffer
	ch := queryPatternsTestHelper("my-org", server.URL, printer.Human, &buf)

	var humanOut bytes.Buffer
	ch.Printer.SetHumanOutput(&humanOut)

	var stdout bytes.Buffer
	cmd := QueryPatternsCmd(ch)
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"download", "my-db", "my-branch", "--output", "-"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(stdout.String(), qt.Equals, queryPatternsCSV)
	c.Assert(humanOut.String(), qt.Equals, "")
	c.Assert(buf.String(), qt.Equals, "")
}

func TestBranch_QueryPatternsDownloadCmd_Failed(t *testing.T) {
	c := qt.New(t)

	prevInterval := queryPatternsPollInterval
	queryPatternsPollInterval = 10 * time.Millisecond
	t.Cleanup(func() { queryPatternsPollInterval = prevInterval })

	server := queryPatternsServer(t, c, []string{"failed"})

	var buf bytes.Buffer
	ch := queryPatternsTestHelper("my-org", server.URL, printer.JSON, &buf)

	cmd := QueryPatternsCmd(ch)
	cmd.SetArgs([]string{"download", "my-db", "my-branch", "--output", filepath.Join(t.TempDir(), "report.csv")})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "failed to generate")
}

func TestBranch_QueryPatternsDownloadCmd_NotFound(t *testing.T) {
	c := qt.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"code":"not_found","message":"Not Found"}`)
	}))
	t.Cleanup(server.Close)

	var buf bytes.Buffer
	ch := queryPatternsTestHelper("my-org", server.URL, printer.JSON, &buf)

	cmd := QueryPatternsCmd(ch)
	cmd.SetArgs([]string{"download", "my-db", "my-branch"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "query insights is not enabled")
}

func testQueryPatternsReport() *ps.QueryPatternsReport {
	return &ps.QueryPatternsReport{
		PublicID: "report1",
		State:    "completed",
		Actor:    &ps.Actor{Name: "Ada"},
	}
}

func TestBranch_QueryPatternsListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.QueryPatternsService{
		ListReportsFn: func(ctx context.Context, req *ps.ListQueryPatternsReportsRequest, opts ...ps.ListOption) ([]*ps.QueryPatternsReport, error) {
			c.Assert(req.Database, qt.Equals, "my-db")
			c.Assert(req.Branch, qt.Equals, "my-branch")
			return []*ps.QueryPatternsReport{testQueryPatternsReport()}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{QueryPatterns: svc}, nil
		},
	}

	cmd := ListQueryPatternsCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListReportsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "report1")
}

func TestBranch_QueryPatternsShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.QueryPatternsService{
		GetReportFn: func(ctx context.Context, req *ps.GetQueryPatternsReportRequest) (*ps.QueryPatternsReport, error) {
			c.Assert(req.Report, qt.Equals, "report1")
			return testQueryPatternsReport(), nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{QueryPatterns: svc}, nil
		},
	}

	cmd := ShowQueryPatternsCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch", "report1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetReportFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"state": "completed"`)
}

func TestBranch_QueryPatternsDeleteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.QueryPatternsService{
		DeleteReportFn: func(ctx context.Context, req *ps.DeleteQueryPatternsReportRequest) error {
			c.Assert(req.Report, qt.Equals, "report1")
			return nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{QueryPatterns: svc}, nil
		},
	}

	cmd := DeleteQueryPatternsCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch", "report1", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.DeleteReportFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "query pattern report deleted")
}
