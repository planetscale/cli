package branch

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

func readOnlyReplicaTestHelper(org string, svc *mock.ReadOnlyReplicasService, format printer.Format, buf *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	if format == printer.Human {
		p.SetHumanOutput(buf)
	}
	p.SetResourceOutput(buf)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{ReadOnlyReplicas: svc}, nil
		},
	}
}

func TestReadOnlyReplicaListCmdJSON(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	replicas := []*ps.ReadOnlyReplica{{
		ID: "replica-1", Name: "analytics", State: "ready", Replicas: 2,
		ClusterDisplayName: "PS-10", AccessHostURL: "host.example.com",
		Ready: true, Region: ps.Region{Slug: "us-east"},
		CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}}
	svc := &mock.ReadOnlyReplicasService{
		ListFn: func(_ context.Context, req *ps.ListReadOnlyReplicasRequest, opts ...ps.ListOption) ([]*ps.ReadOnlyReplica, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "app")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(opts, qt.HasLen, 2)
			return replicas, nil
		},
	}

	cmd := ReadOnlyReplicaListCmd(readOnlyReplicaTestHelper("planetscale", svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"app", "main", "--page", "2", "--per-page", "25"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, replicas)
}

func TestReadOnlyReplicaListCmdHumanEmpty(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.ReadOnlyReplicasService{
		ListFn: func(_ context.Context, _ *ps.ListReadOnlyReplicasRequest, _ ...ps.ListOption) ([]*ps.ReadOnlyReplica, error) {
			return []*ps.ReadOnlyReplica{}, nil
		},
	}
	cmd := ReadOnlyReplicaListCmd(readOnlyReplicaTestHelper("planetscale", svc, printer.Human, &buf))
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(buf.String(), qt.Contains, "No read-only replicas exist in branch")
}

func TestReadOnlyReplicaCreateCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.ReadOnlyReplicasService{
		CreateFn: func(_ context.Context, req *ps.CreateReadOnlyReplicaRequest) (*ps.ReadOnlyReplica, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "app")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.Name, qt.Equals, "analytics")
			c.Assert(req.Region, qt.Equals, "us-east")
			c.Assert(req.ClusterSize, qt.Equals, "PS-10")
			c.Assert(req.Replicas, qt.IsNotNil)
			c.Assert(*req.Replicas, qt.Equals, 3)
			return &ps.ReadOnlyReplica{ID: "replica-1", Name: "analytics"}, nil
		},
	}
	cmd := ReadOnlyReplicaCreateCmd(readOnlyReplicaTestHelper("planetscale", svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"app", "main", "--name", "analytics", "--region", "us-east", "--replicas", "3", "--cluster-size", "PS-10"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &ps.ReadOnlyReplica{ID: "replica-1", Name: "analytics"})
}

func TestReadOnlyReplicaCreateCmdReplicasDefault(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.ReadOnlyReplicasService{
		CreateFn: func(_ context.Context, req *ps.CreateReadOnlyReplicaRequest) (*ps.ReadOnlyReplica, error) {
			c.Assert(req.Replicas, qt.IsNil)
			return &ps.ReadOnlyReplica{Name: req.Name}, nil
		},
	}
	cmd := ReadOnlyReplicaCreateCmd(readOnlyReplicaTestHelper("planetscale", svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"app", "main", "--name", "analytics", "--region", "us-east"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}

func TestReadOnlyReplicaShowCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.ReadOnlyReplicasService{
		GetFn: func(_ context.Context, req *ps.GetReadOnlyReplicaRequest) (*ps.ReadOnlyReplica, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "app")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.Name, qt.Equals, "analytics")
			return &ps.ReadOnlyReplica{ID: "replica-1", Name: "analytics"}, nil
		},
	}
	cmd := ReadOnlyReplicaShowCmd(readOnlyReplicaTestHelper("planetscale", svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"app", "main", "analytics"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &ps.ReadOnlyReplica{ID: "replica-1", Name: "analytics"})
}

func TestReadOnlyReplicaDeleteCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	svc := &mock.ReadOnlyReplicasService{
		DeleteFn: func(_ context.Context, req *ps.DeleteReadOnlyReplicaRequest) error {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "app")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.Name, qt.Equals, "analytics")
			return nil
		},
	}
	cmd := ReadOnlyReplicaDeleteCmd(readOnlyReplicaTestHelper("planetscale", svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"app", "main", "analytics", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{"result": "read-only replica deleted"})
}

func TestReadOnlyReplicaChangesCmd(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	changes := []*ps.ReadOnlyReplicaChangeRequest{{
		ID: "change-1", State: "completed", ClusterDisplayName: "PS-10",
		Replica:   &ps.ReadOnlyReplicaRef{Name: "analytics"},
		CreatedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}}
	svc := &mock.ReadOnlyReplicasService{
		ListChangesFn: func(_ context.Context, req *ps.ListReadOnlyReplicaChangesRequest, opts ...ps.ListOption) ([]*ps.ReadOnlyReplicaChangeRequest, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "app")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.Period, qt.Equals, "1d")
			c.Assert(opts, qt.HasLen, 2)
			return changes, nil
		},
	}
	cmd := ReadOnlyReplicaChangesCmd(readOnlyReplicaTestHelper("planetscale", svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"app", "main", "--period", "1d", "--page", "2", "--per-page", "25"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListChangesFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, changes)
}
