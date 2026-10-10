package branch

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestSnapshot_CreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	svc := &mock.DatabaseBranchesService{
		RefreshSchemaFn: func(ctx context.Context, req *ps.RefreshSchemaRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			return nil
		},
		GetFn: func(ctx context.Context, req *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			return &ps.DatabaseBranch{Name: branch, SchemaReady: true}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: svc,
			}, nil
		},
	}

	cmd := RefreshSchemaCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.RefreshSchemaFnInvoked, qt.IsTrue)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"result": "schema refreshed"`)
}

func TestRefreshSchemaWaitsUntilSchemaReady(t *testing.T) {
	c := qt.New(t)
	restoreSchemaRefreshTiming(t)

	schemaRefreshPollInterval = time.Millisecond
	schemaRefreshTimeout = time.Second

	var buf bytes.Buffer
	var polls int
	svc := &mock.DatabaseBranchesService{
		RefreshSchemaFn: func(context.Context, *ps.RefreshSchemaRequest) error {
			return nil
		},
		GetFn: func(context.Context, *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			polls++
			return &ps.DatabaseBranch{SchemaReady: polls > 1}, nil
		},
	}

	cmd := RefreshSchemaCmd(refreshSchemaHelper(svc, &buf))
	cmd.SetArgs([]string{"planetscale", "development"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(polls, qt.Equals, 2)
	c.Assert(buf.String(), qt.Contains, `"result": "schema refreshed"`)
}

func TestRefreshSchemaReturnsPollError(t *testing.T) {
	c := qt.New(t)
	restoreSchemaRefreshTiming(t)

	schemaRefreshPollInterval = time.Millisecond
	schemaRefreshTimeout = 50 * time.Millisecond

	var buf bytes.Buffer
	svc := &mock.DatabaseBranchesService{
		RefreshSchemaFn: func(context.Context, *ps.RefreshSchemaRequest) error {
			return nil
		},
		GetFn: func(context.Context, *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			return nil, errors.New("connection refused")
		},
	}

	cmd := RefreshSchemaCmd(refreshSchemaHelper(svc, &buf))
	cmd.SetArgs([]string{"planetscale", "development"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "connection refused")
	c.Assert(buf.String(), qt.Equals, "")
}

func TestRefreshSchemaTimesOutWhileSchemaIsRefreshing(t *testing.T) {
	c := qt.New(t)
	restoreSchemaRefreshTiming(t)

	schemaRefreshPollInterval = time.Millisecond
	schemaRefreshTimeout = 50 * time.Millisecond

	var buf bytes.Buffer
	svc := &mock.DatabaseBranchesService{
		RefreshSchemaFn: func(context.Context, *ps.RefreshSchemaRequest) error {
			return nil
		},
		GetFn: func(context.Context, *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			return &ps.DatabaseBranch{SchemaReady: false}, nil
		},
	}

	cmd := RefreshSchemaCmd(refreshSchemaHelper(svc, &buf))
	cmd.SetArgs([]string{"planetscale", "development"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "schema refreshing; please try again in a few moments")
	c.Assert(buf.String(), qt.Equals, "")
}

func refreshSchemaHelper(svc *mock.DatabaseBranchesService, buf *bytes.Buffer) *cmdutil.Helper {
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DatabaseBranches: svc}, nil
		},
	}
}

func restoreSchemaRefreshTiming(t *testing.T) {
	t.Helper()
	interval, timeout := schemaRefreshPollInterval, schemaRefreshTimeout
	t.Cleanup(func() {
		schemaRefreshPollInterval = interval
		schemaRefreshTimeout = timeout
	})
}
