package org

import (
	"bytes"
	"context"
	"testing"
	"testing/fstest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	qt "github.com/frankban/quicktest"
)

func TestOrganization_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"

	svc := &mock.OrganizationsService{
		ListFn: func(ctx context.Context) ([]*ps.Organization, error) {
			return []*ps.Organization{
				{Name: "foo"},
				{Name: "bar"},
			}, nil
		},
	}

	fs := fstest.MapFS{
		".pscale.yml": &fstest.MapFile{
			Data: []byte("org: " + org + "\n"),
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		ConfigFS: config.NewConfigFS(fs),
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Organizations: svc,
			}, nil
		},
	}

	cmd := ListCmd(ch)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)

	orgs := []*organization{
		{Name: "foo", Current: false},
		{Name: "bar", Current: false},
	}
	c.Assert(buf.String(), qt.JSONEquals, orgs)
}

func TestOrganization_ListCmd_WithCurrentOrg(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	currentOrg := "foo"

	svc := &mock.OrganizationsService{
		ListFn: func(ctx context.Context) ([]*ps.Organization, error) {
			return []*ps.Organization{
				{Name: "foo"},
				{Name: "bar"},
			}, nil
		},
	}

	fs := fstest.MapFS{
		".pscale.yml": &fstest.MapFile{
			Data: []byte("org: " + currentOrg + "\n"),
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: currentOrg,
		},
		ConfigFS: config.NewConfigFS(fs),
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Organizations: svc,
			}, nil
		},
	}

	cmd := ListCmd(ch)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)

	orgs := []*organization{
		{Name: "foo", Current: true},
		{Name: "bar", Current: false},
	}
	c.Assert(buf.String(), qt.JSONEquals, orgs)
}

func TestOrganization_ListCmd_HumanFormat(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	currentOrg := "foo"

	svc := &mock.OrganizationsService{
		ListFn: func(ctx context.Context) ([]*ps.Organization, error) {
			return []*ps.Organization{
				{Name: "foo"},
				{Name: "bar"},
			}, nil
		},
	}

	fs := fstest.MapFS{
		".pscale.yml": &fstest.MapFile{
			Data: []byte("org: " + currentOrg + "\n"),
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: currentOrg,
		},
		ConfigFS: config.NewConfigFS(fs),
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Organizations: svc,
			}, nil
		},
	}

	cmd := ListCmd(ch)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)

	// For human format, the current org should have an asterisk prefix
	output := buf.String()
	c.Assert(output, qt.Contains, "* foo")
	c.Assert(output, qt.Contains, "bar")

	c.Assert(output, qt.Not(qt.Contains), "* bar")
}

func TestOrganization_ListCmd_NoConfig(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.OrganizationsService{
		ListFn: func(ctx context.Context) ([]*ps.Organization, error) {
			return []*ps.Organization{
				{Name: "foo"},
				{Name: "bar"},
			}, nil
		},
	}

	// Create an empty filesystem (no config file)
	fs := fstest.MapFS{}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: "",
		},
		ConfigFS: config.NewConfigFS(fs),
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Organizations: svc,
			}, nil
		},
	}

	cmd := ListCmd(ch)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)

	orgs := []*organization{
		{Name: "foo", Current: false},
		{Name: "bar", Current: false},
	}
	c.Assert(buf.String(), qt.JSONEquals, orgs)
}
