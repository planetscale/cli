package keyspace

import (
	"bytes"
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestKeyspace_CreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"

	ks := &ps.Keyspace{
		ID:          "wantid",
		Name:        keyspace,
		ClusterSize: ps.ClusterSize("PS-20"),
		Replicas:    3,
		Shards:      2,
	}

	svc := &mock.BranchKeyspacesService{
		CreateFn: func(ctx context.Context, req *ps.CreateBranchKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, keyspace)
			c.Assert(req.ClusterSize, qt.Equals, ps.ClusterSize("PS-20"))
			c.Assert(req.ExtraReplicas, qt.Equals, 1)
			c.Assert(req.Shards, qt.Equals, 2)

			return ks, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Keyspaces: svc,
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--cluster-size", "PS-20", "--shards", "2", "--additional-replicas", "1"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, ks)
}

func TestKeyspace_CreateCmdOnlyClusterSize(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "unsharded"

	ks := &ps.Keyspace{
		ID:            "wantid",
		Name:          keyspace,
		ClusterSize:   ps.ClusterSize("PS-10"),
		Replicas:      2,
		ExtraReplicas: 0,
		Shards:        1,
	}

	svc := &mock.BranchKeyspacesService{
		CreateFn: func(ctx context.Context, req *ps.CreateBranchKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, keyspace)
			c.Assert(req.ClusterSize, qt.Equals, ps.ClusterSize("PS-10"))
			c.Assert(req.ExtraReplicas, qt.Equals, 0)
			c.Assert(req.Shards, qt.Equals, 1)

			return ks, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Keyspaces: svc,
			}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--cluster-size", "PS-10"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, ks)
}
