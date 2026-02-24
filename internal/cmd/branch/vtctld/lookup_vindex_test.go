package vtctld

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestLookupVindexCreate(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.LookupVindexService{
		CreateFn: func(ctx context.Context, req *ps.LookupVindexCreateRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, "my-vindex")
			c.Assert(req.TableKeyspace, qt.Equals, "my-keyspace")
			return json.RawMessage(`{"name":"my-vindex"}`), nil
		},
	}

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				LookupVindex: svc,
			}, nil
		},
	}

	cmd := LookupVindexCmd(ch)
	cmd.SetArgs([]string{"create", db, branch,
		"--name", "my-vindex",
		"--table-keyspace", "my-keyspace",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}

func TestLookupVindexShow(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.LookupVindexService{
		ShowFn: func(ctx context.Context, req *ps.LookupVindexShowRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, "my-vindex")
			c.Assert(req.TableKeyspace, qt.Equals, "my-keyspace")
			return json.RawMessage(`{"name":"my-vindex"}`), nil
		},
	}

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				LookupVindex: svc,
			}, nil
		},
	}

	cmd := LookupVindexCmd(ch)
	cmd.SetArgs([]string{"show", db, branch,
		"--name", "my-vindex",
		"--table-keyspace", "my-keyspace",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ShowFnInvoked, qt.IsTrue)
}
