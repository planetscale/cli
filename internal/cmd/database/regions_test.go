package database

import (
	"bytes"
	"context"
	"net/url"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func TestDatabase_RegionsListCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	regions := []*ps.Region{{
		ID:                  "reg123",
		Name:                "US East",
		Slug:                "us-east",
		Location:            "Northern Virginia",
		Provider:            "AWS",
		Enabled:             true,
		MySQLSupported:      true,
		PostgreSQLSupported: true,
	}}

	svc := &mock.DatabaseService{
		ListRegionsFn: func(_ context.Context, req *ps.ListDatabaseRegionsRequest, opts ...ps.ListOption) ([]*ps.Region, error) {
			c.Assert(req.Organization, qt.Equals, "acme")
			c.Assert(req.Database, qt.Equals, "app")
			values := url.Values{}
			listOpts := &ps.ListOptions{URLValues: &values}
			for _, opt := range opts {
				c.Assert(opt(listOpts), qt.IsNil)
			}
			c.Assert(values.Get("page"), qt.Equals, "2")
			c.Assert(values.Get("per_page"), qt.Equals, "25")
			return regions, nil
		},
	}

	cmd := RegionsListCmd(databaseRegionsTestHelper(printer.JSON, &out, svc, nil))
	cmd.SetArgs([]string{"app", "--page", "2", "--per-page", "25"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListRegionsFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, regions)
	c.Assert(cmd.Aliases, qt.Contains, "ls")
}

func TestDatabase_RegionsListCmdHuman(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.DatabaseService{
		ListRegionsFn: func(context.Context, *ps.ListDatabaseRegionsRequest, ...ps.ListOption) ([]*ps.Region, error) {
			return []*ps.Region{{
				Name:                "US East",
				Slug:                "us-east",
				Location:            "Northern Virginia",
				Provider:            "AWS",
				Enabled:             true,
				MySQLSupported:      true,
				PostgreSQLSupported: false,
			}}, nil
		},
	}

	cmd := RegionsListCmd(databaseRegionsTestHelper(printer.Human, &out, svc, nil))
	cmd.SetArgs([]string{"app"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "US East")
	c.Assert(out.String(), qt.Contains, "us-east")
	c.Assert(out.String(), qt.Contains, "MYSQL SUPPORTED")
}

func TestDatabase_ReadOnlyRegionsListCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	regions := []*ps.ReadOnlyRegion{{
		ID:          "ror123",
		DisplayName: "Europe West",
		Ready:       true,
		Region: ps.Region{
			Slug:     "eu-west",
			Provider: "AWS",
		},
	}}

	svc := &mock.ReadOnlyRegionsService{
		ListFn: func(_ context.Context, req *ps.ListReadOnlyRegionsRequest, opts ...ps.ListOption) ([]*ps.ReadOnlyRegion, error) {
			c.Assert(req.Organization, qt.Equals, "acme")
			c.Assert(req.Database, qt.Equals, "app")
			values := url.Values{}
			listOpts := &ps.ListOptions{URLValues: &values}
			for _, opt := range opts {
				c.Assert(opt(listOpts), qt.IsNil)
			}
			c.Assert(values.Get("page"), qt.Equals, "3")
			c.Assert(values.Get("per_page"), qt.Equals, "10")
			return regions, nil
		},
	}

	cmd := ReadOnlyRegionsListCmd(databaseRegionsTestHelper(printer.JSON, &out, nil, svc))
	cmd.SetArgs([]string{"app", "--page", "3", "--per-page", "10"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, regions)
	c.Assert(cmd.Aliases, qt.Contains, "ls")
}

func TestDatabase_ReadOnlyRegionsListCmdHuman(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.ReadOnlyRegionsService{
		ListFn: func(context.Context, *ps.ListReadOnlyRegionsRequest, ...ps.ListOption) ([]*ps.ReadOnlyRegion, error) {
			return []*ps.ReadOnlyRegion{{
				ID:          "ror123",
				DisplayName: "Europe West",
				Ready:       true,
				Region: ps.Region{
					Slug:     "eu-west",
					Provider: "AWS",
				},
			}}, nil
		},
	}

	cmd := ReadOnlyRegionsListCmd(databaseRegionsTestHelper(printer.Human, &out, nil, svc))
	cmd.SetArgs([]string{"app"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "ror123")
	c.Assert(out.String(), qt.Contains, "eu-west")
	c.Assert(out.String(), qt.Contains, "Europe West")
}

func TestDatabase_RegionCommandsRegistered(t *testing.T) {
	c := qt.New(t)
	ch := databaseRegionsTestHelper(printer.JSON, &bytes.Buffer{}, &mock.DatabaseService{}, &mock.ReadOnlyRegionsService{})
	cmd := DatabaseCmd(ch)

	found, _, err := cmd.Find([]string{"regions", "list"})
	c.Assert(err, qt.IsNil)
	c.Assert(found.Use, qt.Equals, "list <database>")

	found, _, err = cmd.Find([]string{"read-only-regions", "ls"})
	c.Assert(err, qt.IsNil)
	c.Assert(found.Use, qt.Equals, "list <database>")
}

func databaseRegionsTestHelper(format printer.Format, out *bytes.Buffer, databases ps.DatabasesService, readOnlyRegions ps.ReadOnlyRegionsService) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(out)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:       databases,
				ReadOnlyRegions: readOnlyRegions,
			}, nil
		},
	}
}
