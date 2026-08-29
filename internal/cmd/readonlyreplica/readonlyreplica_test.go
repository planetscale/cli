package readonlyreplica

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

func testReplica() *ps.PostgresReadOnlyReplica {
	readyAt := time.Date(2026, 8, 28, 10, 20, 23, 0, time.UTC)
	return &ps.PostgresReadOnlyReplica{
		ID:                 "replica-1",
		Name:               "analytics",
		State:              "ready",
		Replicas:           2,
		ClusterName:        "PS_10_GCP_X86",
		ClusterDisplayName: "PS-10",
		AccessHostURL:      "replica.example.com",
		CreatedAt:          time.Date(2026, 8, 28, 10, 19, 23, 0, time.UTC),
		UpdatedAt:          readyAt,
		ReadyAt:            &readyAt,
		Ready:              true,
		Actor:              ps.Actor{ID: "user-1", Name: "Alice"},
		Region:             ps.Region{ID: "region-1", Slug: "us-east", Name: "US East"},
	}
}

func testHelper(org string, dbSvc *mock.DatabaseService, replicaSvc *mock.PostgresReadOnlyReplicasService, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases:                dbSvc,
				PostgresReadOnlyReplicas: replicaSvc,
			}, nil
		},
	}
}

func postgresDatabase(name string) *ps.Database {
	return &ps.Database{Name: name, Kind: ps.DatabaseEnginePostgres}
}

func databaseService(c *qt.C, org, database string) *mock.DatabaseService {
	return &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, database)
			return postgresDatabase(database), nil
		},
	}
}

func TestListCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	org, database, branch := "planetscale", "mydb", "main"
	replica := testReplica()
	svc := &mock.PostgresReadOnlyReplicasService{
		ListFn: func(ctx context.Context, req *ps.ListPostgresReadOnlyReplicasRequest) ([]*ps.PostgresReadOnlyReplica, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, database)
			c.Assert(req.Branch, qt.Equals, branch)
			return []*ps.PostgresReadOnlyReplica{replica}, nil
		},
	}

	cmd := ListCmd(testHelper(org, databaseService(c, org, database), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{database, branch})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, []*ReadOnlyReplica{{orig: replica}})
}

func TestShowCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	org, database, branch := "planetscale", "mydb", "main"
	replica := testReplica()
	svc := &mock.PostgresReadOnlyReplicasService{
		GetFn: func(ctx context.Context, req *ps.GetPostgresReadOnlyReplicaRequest) (*ps.PostgresReadOnlyReplica, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, database)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Replica, qt.Equals, "analytics")
			return replica, nil
		},
	}

	cmd := ShowCmd(testHelper(org, databaseService(c, org, database), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{database, branch, "analytics"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &ReadOnlyReplica{orig: replica})
}

func TestCreateCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	org, database, branch := "planetscale", "mydb", "main"
	replica := testReplica()
	svc := &mock.PostgresReadOnlyReplicasService{
		CreateFn: func(ctx context.Context, req *ps.CreatePostgresReadOnlyReplicaRequest) (*ps.PostgresReadOnlyReplica, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, database)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, "analytics")
			c.Assert(req.Region, qt.Equals, "us-east")
			c.Assert(req.ClusterSize, qt.Equals, "PS_10_GCP_X86")
			c.Assert(req.Replicas, qt.IsNotNil)
			c.Assert(*req.Replicas, qt.Equals, 2)
			return replica, nil
		},
	}

	cmd := CreateCmd(testHelper(org, databaseService(c, org, database), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{database, branch, "analytics", "--region", "us-east", "--replicas", "2", "--cluster-size", "PS_10_GCP_X86"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &ReadOnlyReplica{orig: replica})
}

func TestUpdateCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	org, database, branch := "planetscale", "mydb", "main"
	replica := testReplica()
	svc := &mock.PostgresReadOnlyReplicasService{
		UpdateFn: func(ctx context.Context, req *ps.UpdatePostgresReadOnlyReplicaRequest) (*ps.PostgresReadOnlyReplica, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, database)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Replica, qt.Equals, "analytics")
			c.Assert(req.ClusterSize, qt.Equals, "PS_20_GCP_X86")
			c.Assert(req.Replicas, qt.IsNotNil)
			c.Assert(*req.Replicas, qt.Equals, 3)
			c.Assert(req.Parameters, qt.DeepEquals, map[string]map[string]string{
				"pgconf": {"max_connections": "300"},
			})
			return replica, nil
		},
	}

	cmd := UpdateCmd(testHelper(org, databaseService(c, org, database), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{
		database, branch, "analytics",
		"--replicas", "3",
		"--cluster-size", "PS_20_GCP_X86",
		"--parameters", "pgconf.max_connections=300",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &ReadOnlyReplica{orig: replica})
}

func TestUpdateCmdRequiresChange(t *testing.T) {
	c := qt.New(t)
	svc := &mock.PostgresReadOnlyReplicasService{}
	cmd := UpdateCmd(testHelper("planetscale", &mock.DatabaseService{}, svc, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "analytics"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `nothing to change:.*`)
	c.Assert(svc.UpdateFnInvoked, qt.IsFalse)
}

func TestDeleteCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	org, database, branch := "planetscale", "mydb", "main"
	svc := &mock.PostgresReadOnlyReplicasService{
		DeleteFn: func(ctx context.Context, req *ps.DeletePostgresReadOnlyReplicaRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, database)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Replica, qt.Equals, "analytics")
			return nil
		},
	}

	cmd := DeleteCmd(testHelper(org, databaseService(c, org, database), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{database, branch, "analytics", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{
		"result":   "read-only replica deleted",
		"name":     "analytics",
		"database": database,
		"branch":   branch,
	})
}

func TestDeleteCmdRequiresForceInJSON(t *testing.T) {
	c := qt.New(t)
	org, database := "planetscale", "mydb"
	svc := &mock.PostgresReadOnlyReplicasService{}
	cmd := DeleteCmd(testHelper(org, databaseService(c, org, database), svc, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{database, "main", "analytics"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `(?s).*run with --force.*`)
	c.Assert(svc.DeleteFnInvoked, qt.IsFalse)
}

func TestListCmdRejectsMySQL(t *testing.T) {
	c := qt.New(t)
	org, database := "planetscale", "mydb"
	dbSvc := &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return &ps.Database{Name: database, Kind: ps.DatabaseEngineMySQL}, nil
		},
	}
	svc := &mock.PostgresReadOnlyReplicasService{}
	cmd := ListCmd(testHelper(org, dbSvc, svc, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{database, "main"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `(?s).*only available for PostgreSQL.*mysql.*`)
	c.Assert(svc.ListFnInvoked, qt.IsFalse)
}

func TestParseParameters(t *testing.T) {
	c := qt.New(t)
	parameters, err := parseParameters([]string{"pgconf.max_connections=300", "pgconf.work_mem=64MB"})
	c.Assert(err, qt.IsNil)
	c.Assert(parameters, qt.DeepEquals, map[string]map[string]string{
		"pgconf": {
			"max_connections": "300",
			"work_mem":        "64MB",
		},
	})

	_, err = parseParameters([]string{"max_connections=300"})
	c.Assert(err, qt.ErrorMatches, `invalid --parameters.*`)
}
