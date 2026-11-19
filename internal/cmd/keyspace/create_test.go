package keyspace

import (
	"bytes"
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
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

	maxRollout := 8
	ks := &ps.Keyspace{
		ID:          "wantid",
		Name:        keyspace,
		ClusterSize: "PS-20",
		Replicas:    3,
		Shards:      2,
		MaxRollout:  &maxRollout,
	}

	svc := &mock.KeyspacesService{
		CreateFn: func(ctx context.Context, req *ps.CreateKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, keyspace)
			c.Assert(req.ClusterSize, qt.Equals, "PS-20")
			c.Assert(req.ExtraReplicas, qt.Equals, 1)
			c.Assert(req.Shards, qt.Equals, 2)
			c.Assert(req.MaxRollout, qt.Not(qt.IsNil))
			c.Assert(*req.MaxRollout, qt.Equals, 8)

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
	cmd.SetArgs([]string{db, branch, keyspace, "--cluster-size", "PS-20", "--shards", "2", "--additional-replicas", "1", "--max-rollout", "8"})
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(stderr.String(), qt.Equals, "")
	c.Assert(buf.String(), qt.JSONEquals, ks)
}

func TestKeyspace_CreateCmdWithWaitPrintsReadyKeyspace(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"
	pending := &ps.Keyspace{Name: keyspace, Ready: false}
	ready := &ps.Keyspace{Name: keyspace, Ready: true}

	svc := &mock.KeyspacesService{
		CreateFn: func(_ context.Context, _ *ps.CreateKeyspaceRequest) (*ps.Keyspace, error) {
			return pending, nil
		},
		GetFn: func(_ context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req, qt.DeepEquals, &ps.GetKeyspaceRequest{
				Organization: org,
				Database:     db,
				Branch:       branch,
				Keyspace:     keyspace,
			})
			return ready, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}
	debug := false
	ch.SetDebug(&debug)

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--cluster-size", "PS-20", "--wait"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, ready)
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
		ClusterSize:   "PS-10",
		Replicas:      2,
		ExtraReplicas: 0,
		Shards:        1,
	}

	svc := &mock.KeyspacesService{
		CreateFn: func(ctx context.Context, req *ps.CreateKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, keyspace)
			c.Assert(req.ClusterSize, qt.Equals, "PS-10")
			c.Assert(req.ExtraReplicas, qt.Equals, 0)
			c.Assert(req.Shards, qt.Equals, 1)
			c.Assert(req.MaxRollout, qt.IsNil)

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

func TestKeyspace_CreateCmdRejectsOutOfRangeMaxRollout(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	for _, arg := range []string{"--max-rollout=0", "--max-rollout=33"} {
		svc := &mock.KeyspacesService{}
		ch := &cmdutil.Helper{
			Printer: p,
			Config:  &config.Config{Organization: "planetscale"},
			Client: func() (*ps.Client, error) {
				return &ps.Client{Keyspaces: svc}, nil
			},
		}

		cmd := CreateCmd(ch)
		cmd.SetArgs([]string{"planetscale", "main", "sharded", "--cluster-size", "PS-20", arg})
		err := cmd.Execute()
		c.Assert(err, qt.ErrorMatches, ".*max-rollout must be between 1 and 32")
		c.Assert(svc.CreateFnInvoked, qt.IsFalse)
	}
}

func TestKeyspace_CreateCmdWarnsWhenMaxRolloutNotApplied(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&bytes.Buffer{})

	svc := &mock.KeyspacesService{
		CreateFn: func(_ context.Context, req *ps.CreateKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{Name: req.Name}, nil
		},
	}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "sharded", "--cluster-size", "PS-20", "--max-rollout", "8"})
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(stderr.String(), qt.Contains, "--max-rollout was not applied")
}
