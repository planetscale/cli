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
			c.Assert(req.IgnoreNulls, qt.IsNil)
			c.Assert(req.TabletTypesInPreferenceOrder, qt.IsNil)
			c.Assert(req.ContinueAfterCopyWithOwner, qt.IsNil)
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

func TestLookupVindexCreateWithExplicitFalseBoolFlags(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.LookupVindexService{
		CreateFn: func(ctx context.Context, req *ps.LookupVindexCreateRequest) (json.RawMessage, error) {
			c.Assert(req.Name, qt.Equals, "my-vindex")
			c.Assert(req.TableKeyspace, qt.Equals, "my-keyspace")
			c.Assert(req.IgnoreNulls, qt.IsNotNil)
			c.Assert(*req.IgnoreNulls, qt.IsFalse)
			c.Assert(req.TabletTypesInPreferenceOrder, qt.IsNotNil)
			c.Assert(*req.TabletTypesInPreferenceOrder, qt.IsFalse)
			c.Assert(req.ContinueAfterCopyWithOwner, qt.IsNotNil)
			c.Assert(*req.ContinueAfterCopyWithOwner, qt.IsFalse)
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
		"--ignore-nulls=false",
		"--tablet-types-in-preference-order=false",
		"--continue-after-copy-with-owner=false",
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

func TestLookupVindexExternalize(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.LookupVindexService{
		ExternalizeFn: func(ctx context.Context, req *ps.LookupVindexExternalizeRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, "my-vindex")
			c.Assert(req.TableKeyspace, qt.Equals, "my-keyspace")
			c.Assert(req.Delete, qt.IsNil)
			return json.RawMessage(`{"summary":"externalized"}`), nil
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
	cmd.SetArgs([]string{"externalize", db, branch,
		"--name", "my-vindex",
		"--table-keyspace", "my-keyspace",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ExternalizeFnInvoked, qt.IsTrue)
}

func TestLookupVindexExternalizeWithDeleteFalse(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.LookupVindexService{
		ExternalizeFn: func(ctx context.Context, req *ps.LookupVindexExternalizeRequest) (json.RawMessage, error) {
			c.Assert(req.Name, qt.Equals, "my-vindex")
			c.Assert(req.TableKeyspace, qt.Equals, "my-keyspace")
			c.Assert(req.Keyspace, qt.Equals, "lookup-ks")
			c.Assert(req.Delete, qt.IsNotNil)
			c.Assert(*req.Delete, qt.IsFalse)
			return json.RawMessage(`{"summary":"externalized"}`), nil
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
	cmd.SetArgs([]string{"externalize", db, branch,
		"--name", "my-vindex",
		"--table-keyspace", "my-keyspace",
		"--keyspace", "lookup-ks",
		"--delete=false",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ExternalizeFnInvoked, qt.IsTrue)
}

func TestLookupVindexInternalize(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.LookupVindexService{
		InternalizeFn: func(ctx context.Context, req *ps.LookupVindexInternalizeRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, "my-vindex")
			c.Assert(req.TableKeyspace, qt.Equals, "my-keyspace")
			c.Assert(req.Keyspace, qt.Equals, "lookup-ks")
			return json.RawMessage(`{"summary":"internalized"}`), nil
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
	cmd.SetArgs([]string{"internalize", db, branch,
		"--name", "my-vindex",
		"--table-keyspace", "my-keyspace",
		"--keyspace", "lookup-ks",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.InternalizeFnInvoked, qt.IsTrue)
}

func TestLookupVindexCancel(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.LookupVindexService{
		CancelFn: func(ctx context.Context, req *ps.LookupVindexCancelRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, "my-vindex")
			c.Assert(req.TableKeyspace, qt.Equals, "my-keyspace")
			return json.RawMessage(`{"summary":"canceled"}`), nil
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
	cmd.SetArgs([]string{"cancel", db, branch,
		"--name", "my-vindex",
		"--table-keyspace", "my-keyspace",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CancelFnInvoked, qt.IsTrue)
}

func TestLookupVindexComplete(t *testing.T) {
	c := qt.New(t)

	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	svc := &mock.LookupVindexService{
		CompleteFn: func(ctx context.Context, req *ps.LookupVindexCompleteRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, "my-vindex")
			c.Assert(req.TableKeyspace, qt.Equals, "my-keyspace")
			c.Assert(req.Keyspace, qt.Equals, "lookup-ks")
			return json.RawMessage(`{"summary":"completed"}`), nil
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
	cmd.SetArgs([]string{"complete", db, branch,
		"--name", "my-vindex",
		"--table-keyspace", "my-keyspace",
		"--keyspace", "lookup-ks",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CompleteFnInvoked, qt.IsTrue)
}
