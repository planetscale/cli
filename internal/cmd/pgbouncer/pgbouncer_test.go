package pgbouncer

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

func testBouncer() *ps.PostgresBouncer {
	return &ps.PostgresBouncer{
		ID:   "bouncer-1",
		Name: "read-pool",
		SKU: &ps.PostgresBouncerSKU{
			Name:        "PGB_10",
			DisplayName: "PS-10",
			CPU:         "0.25",
			RAM:         268435456,
		},
		Target:          "replica",
		ReplicasPerCell: 1,
		CreatedAt:       time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt:       time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		Actor:           ps.Actor{ID: "user-1", Name: "Alice"},
		Branch:          ps.PostgresBouncerBranch{ID: "branch-1", Name: "main"},
	}
}

func testHelper(org string, dbSvc *mock.DatabaseService, svc *mock.PostgresBouncersService, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:        dbSvc,
				PostgresBouncers: svc,
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

func TestPgBouncer_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	branch := "main"
	bouncer := testBouncer()

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return postgresDB(db), nil
		},
	}
	svc := &mock.PostgresBouncersService{
		ListFn: func(ctx context.Context, req *ps.ListPostgresBouncersRequest, opts ...ps.ListOption) ([]*ps.PostgresBouncer, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			return []*ps.PostgresBouncer{bouncer}, nil
		},
	}

	cmd := ListCmd(testHelper(org, dbSvc, svc, printer.JSON, &buf))
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(dbSvc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, []*PgBouncer{{orig: bouncer}})
}

func TestPgBouncer_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	branch := "main"
	bouncer := testBouncer()

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB(db), nil
		},
	}
	svc := &mock.PostgresBouncersService{
		GetFn: func(ctx context.Context, req *ps.GetPostgresBouncerRequest) (*ps.PostgresBouncer, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Bouncer, qt.Equals, "read-pool")
			return bouncer, nil
		},
	}

	cmd := ShowCmd(testHelper(org, dbSvc, svc, printer.JSON, &buf))
	cmd.SetArgs([]string{db, branch, "read-pool"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &PgBouncer{orig: bouncer})
}

func TestPgBouncer_CreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	branch := "main"
	bouncer := testBouncer()

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB(db), nil
		},
	}
	svc := &mock.PostgresBouncersService{
		CreateFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRequest) (*ps.PostgresBouncer, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, "read-pool")
			c.Assert(req.Target, qt.Equals, "replica")
			c.Assert(req.BouncerSize, qt.Equals, "PGB_10")
			c.Assert(req.ReplicasPerCell, qt.IsNotNil)
			c.Assert(*req.ReplicasPerCell, qt.Equals, 2)
			return bouncer, nil
		},
	}

	cmd := CreateCmd(testHelper(org, dbSvc, svc, printer.JSON, &buf))
	cmd.SetArgs([]string{
		db, branch,
		"--name", "read-pool",
		"--target", "replica",
		"--size", "PGB_10",
		"--replicas-per-cell", "2",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &PgBouncer{orig: bouncer})
}

func TestPgBouncer_CreateCmd_RequiresTarget(t *testing.T) {
	c := qt.New(t)

	cmd := CreateCmd(testHelper("planetscale", &mock.DatabaseService{}, &mock.PostgresBouncersService{}, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "--name", "read-pool"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "required flag")
}

func TestPgBouncer_DeleteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	branch := "main"
	name := "read-pool"

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB(db), nil
		},
	}
	svc := &mock.PostgresBouncersService{
		DeleteFn: func(ctx context.Context, req *ps.DeletePostgresBouncerRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Bouncer, qt.Equals, name)
			return nil
		},
	}

	cmd := DeleteCmd(testHelper(org, dbSvc, svc, printer.JSON, &buf))
	cmd.SetArgs([]string{db, branch, name, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{
		"result":   "PgBouncer deleted",
		"name":     name,
		"database": db,
		"branch":   branch,
	})
}

func TestPgBouncer_DeleteCmd_RequiresForceInJSON(t *testing.T) {
	c := qt.New(t)

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB("mydb"), nil
		},
	}
	svc := &mock.PostgresBouncersService{}
	cmd := DeleteCmd(testHelper("planetscale", dbSvc, svc, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "read-pool"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*run with --force.*`)
	c.Assert(svc.DeleteFnInvoked, qt.IsFalse)
}

func TestPgBouncer_ListCmd_RejectsMySQL(t *testing.T) {
	c := qt.New(t)

	org := "planetscale"
	db := "mysql-db"
	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Database, qt.Equals, db)
			return mysqlDB(db), nil
		},
	}
	svc := &mock.PostgresBouncersService{}

	cmd := ListCmd(testHelper(org, dbSvc, svc, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{db, "main"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*only available for PostgreSQL.*mysql.*`)
	c.Assert(dbSvc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.ListFnInvoked, qt.IsFalse)
}

func testResize() *ps.PostgresBouncerResizeRequest {
	return &ps.PostgresBouncerResizeRequest{
		ID:                      "resize-1",
		State:                   ps.PostgresBouncerResizeStatePending,
		ReplicasPerCell:         2,
		Target:                  "replica",
		PreviousReplicasPerCell: 1,
		PreviousTarget:          "replica",
		CreatedAt:               time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt:               time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		Actor:                   ps.Actor{ID: "user-1", Name: "Alice"},
		Bouncer:                 ps.PostgresBouncerBranch{ID: "bouncer-1", Name: "read-pool"},
		SKU: &ps.PostgresBouncerSKU{
			Name:        "PGB_10",
			DisplayName: "PS-10",
		},
		PreviousSKU: &ps.PostgresBouncerSKU{
			Name:        "PGB_5",
			DisplayName: "PS-5",
		},
	}
}

func TestPgBouncer_ResizeCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	branch := "main"
	name := "read-pool"
	resize := testResize()

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB(db), nil
		},
	}
	svc := &mock.PostgresBouncersService{
		ResizeFn: func(ctx context.Context, req *ps.ResizePostgresBouncerRequest) (*ps.PostgresBouncerResizeRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Bouncer, qt.Equals, name)
			c.Assert(req.BouncerSize, qt.Equals, "PGB_10")
			c.Assert(req.ReplicasPerCell, qt.IsNotNil)
			c.Assert(*req.ReplicasPerCell, qt.Equals, 2)
			c.Assert(req.Parameters["pgbouncer"]["default_pool_size"], qt.Equals, "100")
			return resize, nil
		},
	}

	cmd := ResizeCmd(testHelper(org, dbSvc, svc, printer.JSON, &buf))
	cmd.SetArgs([]string{
		db, branch, name,
		"--size", "PGB_10",
		"--replicas-per-cell", "2",
		"--parameters", "pgbouncer.default_pool_size=100",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ResizeFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &PgBouncerResize{orig: resize})
}

func TestPgBouncer_ResizeCmd_RequiresChange(t *testing.T) {
	c := qt.New(t)

	cmd := ResizeCmd(testHelper("planetscale", &mock.DatabaseService{}, &mock.PostgresBouncersService{}, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "read-pool"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*nothing to change.*`)
}

func TestPgBouncer_ResizeStatusCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	branch := "main"
	name := "read-pool"
	resize := testResize()

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB(db), nil
		},
	}
	svc := &mock.PostgresBouncersService{
		ListResizesFn: func(ctx context.Context, req *ps.ListPostgresBouncerResizesRequest, opts ...ps.ListOption) ([]*ps.PostgresBouncerResizeRequest, error) {
			c.Assert(req.Bouncer, qt.Equals, name)
			return []*ps.PostgresBouncerResizeRequest{resize}, nil
		},
	}

	cmd := ResizeStatusCmd(testHelper(org, dbSvc, svc, printer.JSON, &buf))
	cmd.SetArgs([]string{db, branch, name})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListResizesFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &PgBouncerResize{orig: resize})
}

func TestPgBouncer_ResizeCancelCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	branch := "main"
	name := "read-pool"

	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB(db), nil
		},
	}
	svc := &mock.PostgresBouncersService{
		CancelResizesFn: func(ctx context.Context, req *ps.CancelPostgresBouncerResizesRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Bouncer, qt.Equals, name)
			return nil
		},
	}

	cmd := ResizeCancelCmd(testHelper(org, dbSvc, svc, printer.JSON, &buf))
	cmd.SetArgs([]string{db, branch, name})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CancelResizesFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{
		"result":   "canceled",
		"name":     name,
		"database": db,
		"branch":   branch,
	})
}
