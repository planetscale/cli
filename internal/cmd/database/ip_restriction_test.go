package database

import (
	"bytes"
	"context"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func ipRestrictionTestEntry() *ps.PostgresCIDR {
	desc := "office"
	return &ps.PostgresCIDR{
		ID:          "cidr-1",
		Schema:      "public",
		Role:        "reader",
		CIDRs:       []string{"192.168.1.0/24", "10.0.0.1/32"},
		Description: &desc,
		CreatedAt:   time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt:   time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		Actor:       ps.Actor{ID: "user-1", Name: "Alice"},
	}
}

func ipRestrictionHelper(c *qt.C, org string, dbSvc *mock.DatabaseService, cidrSvc *mock.PostgresCIDRsService, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:     dbSvc,
				PostgresCIDRs: cidrSvc,
			}, nil
		},
	}
}

func postgresDB(name string) *ps.Database {
	return &ps.Database{Name: name, Kind: ps.DatabaseEnginePostgres}
}

func mysqlDB(name string) *ps.Database {
	return &ps.Database{Name: name, Kind: ps.DatabaseEngineMySQL}
}

func TestIPRestriction_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	entry := ipRestrictionTestEntry()

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return postgresDB(db), nil
		},
	}
	cidrSvc := &mock.PostgresCIDRsService{
		ListFn: func(ctx context.Context, req *ps.ListPostgresCIDRsRequest, opts ...ps.ListOption) ([]*ps.PostgresCIDR, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return []*ps.PostgresCIDR{entry}, nil
		},
	}

	cmd := IPRestrictionListCmd(ipRestrictionHelper(c, org, dbSvc, cidrSvc, printer.JSON, &buf))
	cmd.SetArgs([]string{db})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(dbSvc.GetFnInvoked, qt.IsTrue)
	c.Assert(cidrSvc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, []*IPRestrictionEntry{{orig: entry}})
}

func TestIPRestriction_ListCmd_RejectsMySQL(t *testing.T) {
	c := qt.New(t)

	org := "planetscale"
	db := "mysql-db"
	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return mysqlDB(db), nil
		},
	}
	cidrSvc := &mock.PostgresCIDRsService{}

	cmd := IPRestrictionListCmd(ipRestrictionHelper(c, org, dbSvc, cidrSvc, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{db})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*only available for PostgreSQL.*mysql.*`)
	c.Assert(cidrSvc.ListFnInvoked, qt.IsFalse)
}

func TestIPRestriction_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	entry := ipRestrictionTestEntry()

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB(db), nil
		},
	}
	cidrSvc := &mock.PostgresCIDRsService{
		GetFn: func(ctx context.Context, req *ps.GetPostgresCIDRRequest) (*ps.PostgresCIDR, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.ID, qt.Equals, entry.ID)
			return entry, nil
		},
	}

	cmd := IPRestrictionShowCmd(ipRestrictionHelper(c, org, dbSvc, cidrSvc, printer.JSON, &buf))
	cmd.SetArgs([]string{db, entry.ID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(cidrSvc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &IPRestrictionEntry{orig: entry})
}

func TestIPRestriction_CreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	entry := ipRestrictionTestEntry()

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB(db), nil
		},
	}
	cidrSvc := &mock.PostgresCIDRsService{
		CreateFn: func(ctx context.Context, req *ps.CreatePostgresCIDRRequest) (*ps.PostgresCIDR, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Schema, qt.Equals, "public")
			c.Assert(req.Role, qt.Equals, "reader")
			c.Assert(req.CIDRs, qt.DeepEquals, []string{"192.168.1.0/24", "10.0.0.1/32"})
			c.Assert(req.Description, qt.Equals, "office")
			return entry, nil
		},
	}

	cmd := IPRestrictionCreateCmd(ipRestrictionHelper(c, org, dbSvc, cidrSvc, printer.JSON, &buf))
	cmd.SetArgs([]string{
		db,
		"--cidrs", "192.168.1.0/24,10.0.0.1/32",
		"--schema", "public",
		"--role", "reader",
		"--description", "office",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(cidrSvc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &IPRestrictionEntry{orig: entry})
}

func TestIPRestriction_CreateCmd_RequiresCIDRs(t *testing.T) {
	c := qt.New(t)

	cmd := IPRestrictionCreateCmd(ipRestrictionHelper(c, "planetscale", &mock.DatabaseService{}, &mock.PostgresCIDRsService{}, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "required flag")
}

func TestIPRestriction_UpdateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	entry := ipRestrictionTestEntry()
	entry.CIDRs = []string{"10.0.0.0/8"}

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB(db), nil
		},
	}
	cidrSvc := &mock.PostgresCIDRsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdatePostgresCIDRRequest) (*ps.PostgresCIDR, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.ID, qt.Equals, "cidr-1")
			c.Assert(req.CIDRs, qt.DeepEquals, []string{"10.0.0.0/8"})
			c.Assert(req.Description, qt.IsNotNil)
			c.Assert(*req.Description, qt.Equals, "updated")
			c.Assert(req.Schema, qt.IsNil)
			return entry, nil
		},
	}

	cmd := IPRestrictionUpdateCmd(ipRestrictionHelper(c, org, dbSvc, cidrSvc, printer.JSON, &buf))
	cmd.SetArgs([]string{db, "cidr-1", "--cidrs", "10.0.0.0/8", "--description", "updated"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(cidrSvc.UpdateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &IPRestrictionEntry{orig: entry})
}

func TestIPRestriction_UpdateCmd_RequiresFlag(t *testing.T) {
	c := qt.New(t)

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB("mydb"), nil
		},
	}

	cmd := IPRestrictionUpdateCmd(ipRestrictionHelper(c, "planetscale", dbSvc, &mock.PostgresCIDRsService{}, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "cidr-1"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*at least one of --cidrs.*`)
}

func TestIPRestriction_DeleteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	entryID := "cidr-1"

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB(db), nil
		},
	}
	cidrSvc := &mock.PostgresCIDRsService{
		DeleteFn: func(ctx context.Context, req *ps.DeletePostgresCIDRRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.ID, qt.Equals, entryID)
			return nil
		},
	}

	cmd := IPRestrictionDeleteCmd(ipRestrictionHelper(c, org, dbSvc, cidrSvc, printer.JSON, &buf))
	cmd.SetArgs([]string{db, entryID, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(cidrSvc.DeleteFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{
		"result": "IP restriction entry deleted",
		"id":     entryID,
	})
}

func TestIPRestriction_DeleteCmd_RequiresForceInJSON(t *testing.T) {
	c := qt.New(t)

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB("mydb"), nil
		},
	}
	cidrSvc := &mock.PostgresCIDRsService{}

	cmd := IPRestrictionDeleteCmd(ipRestrictionHelper(c, "planetscale", dbSvc, cidrSvc, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "cidr-1"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*run with --force.*`)
	c.Assert(cidrSvc.DeleteFnInvoked, qt.IsFalse)
}
