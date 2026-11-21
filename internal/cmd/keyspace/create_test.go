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

	ks := &ps.Keyspace{
		ID:          "wantid",
		Name:        keyspace,
		ClusterSize: "PS-20",
		Replicas:    3,
		Shards:      2,
	}

	svc := &mock.KeyspacesService{
		CreateFn: func(ctx context.Context, req *ps.CreateKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, keyspace)
			c.Assert(req.ClusterSize, qt.Equals, "PS-20")
			c.Assert(req.ExtraReplicas, qt.Equals, 1)
			c.Assert(req.Shards, qt.DeepEquals, intPtr(2))
			c.Assert(req.CustomShards, qt.IsNil)

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
		CreateFn: func(_ context.Context, req *ps.CreateKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Shards, qt.DeepEquals, intPtr(1))
			c.Assert(req.CustomShards, qt.IsNil)
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
			c.Assert(req.Shards, qt.DeepEquals, intPtr(1))
			c.Assert(req.CustomShards, qt.IsNil)

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

func intPtr(n int) *int { return &n }

func newCreateTestHelper(c *qt.C, buf *bytes.Buffer, svc *mock.KeyspacesService) *cmdutil.Helper {
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}
	debug := false
	ch.SetDebug(&debug)
	return ch
}

func TestKeyspace_CreateCmdCustomShards(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	ks := &ps.Keyspace{ID: "wantid", Name: "events", ClusterSize: "PS_40", Shards: 3, Sharded: true}

	svc := &mock.KeyspacesService{
		CreateFn: func(_ context.Context, req *ps.CreateKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "planetscale")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.Name, qt.Equals, "events")
			c.Assert(req.ClusterSize, qt.Equals, "PS_40")
			c.Assert(req.ExtraReplicas, qt.Equals, 0)
			c.Assert(req.Shards, qt.IsNil)
			c.Assert(req.CustomShards, qt.DeepEquals, []ps.CreateKeyspaceCustomShard{
				{KeyRange: "-40", ClusterSize: "PS_80"},
				{KeyRange: "40-80"},
				{KeyRange: "80-", ClusterSize: "PS_20"},
			})
			return ks, nil
		},
	}
	ch := newCreateTestHelper(c, &buf, svc)

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{
		"planetscale", "main", "events",
		"--cluster-size", "PS_40",
		"--custom-shard", "key-range=-40,cluster-size=PS-80",
		"--custom-shard", "key-range=40-80",
		"--custom-shard", " key-range = 80- , cluster-size = PS_20 ",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, ks)
}

func TestKeyspace_CreateCmdCustomShardsWithWait(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	pending := &ps.Keyspace{Name: "events", Ready: false}
	ready := &ps.Keyspace{Name: "events", Ready: true, Shards: 2, Sharded: true}

	svc := &mock.KeyspacesService{
		CreateFn: func(_ context.Context, req *ps.CreateKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Shards, qt.IsNil)
			c.Assert(req.CustomShards, qt.DeepEquals, []ps.CreateKeyspaceCustomShard{
				{KeyRange: "-80"},
				{KeyRange: "80-"},
			})
			return pending, nil
		},
		GetFn: func(_ context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Keyspace, qt.Equals, "events")
			return ready, nil
		},
	}
	ch := newCreateTestHelper(c, &buf, svc)

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{
		"planetscale", "main", "events",
		"--cluster-size", "PS_40",
		"--custom-shard", "key-range=-80",
		"--custom-shard", "key-range=80-",
		"--wait",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, ready)
}

func TestKeyspace_CreateCmdCustomShardsErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "shards conflict",
			args: []string{"--shards", "3", "--custom-shard", "key-range=-80", "--custom-shard", "key-range=80-"},
			want: `--shards cannot be combined with --custom-shard`,
		},
		{
			name: "shards conflict with default value set explicitly",
			args: []string{"--shards", "1", "--custom-shard", "key-range=-80", "--custom-shard", "key-range=80-"},
			want: `--shards cannot be combined with --custom-shard`,
		},
		{
			name: "fewer than two shards",
			args: []string{"--custom-shard", "key-range=-"},
			want: `--custom-shard requires at least two shards, got 1`,
		},
		{
			name: "missing key-range",
			args: []string{"--custom-shard", "cluster-size=PS_80", "--custom-shard", "key-range=80-"},
			want: `--custom-shard "cluster-size=PS_80" is missing required field key-range`,
		},
		{
			name: "empty entry",
			args: []string{"--custom-shard", "", "--custom-shard", "key-range=80-"},
			want: `--custom-shard "" is missing required field key-range`,
		},
		{
			name: "unknown field",
			args: []string{"--custom-shard", "key-range=-80,replicas=2", "--custom-shard", "key-range=80-"},
			want: `unknown --custom-shard field "replicas" in "key-range=-80,replicas=2": valid fields are key-range, cluster-size`,
		},
		{
			name: "field without value separator",
			args: []string{"--custom-shard", "-80", "--custom-shard", "key-range=80-"},
			want: `invalid --custom-shard "-80": expected key-range=<range>\[,cluster-size=<size>\]`,
		},
		{
			name: "empty key-range",
			args: []string{"--custom-shard", "key-range=", "--custom-shard", "key-range=80-"},
			want: `invalid --custom-shard "key-range=": key-range must not be empty`,
		},
		{
			name: "empty cluster-size",
			args: []string{"--custom-shard", "key-range=-80,cluster-size=", "--custom-shard", "key-range=80-"},
			want: `invalid --custom-shard "key-range=-80,cluster-size=": cluster-size must not be empty`,
		},
		{
			name: "duplicate field",
			args: []string{"--custom-shard", "key-range=-80,key-range=80-", "--custom-shard", "key-range=80-"},
			want: `duplicate --custom-shard field "key-range" in "key-range=-80,key-range=80-"`,
		},
		{
			name: "duplicate cluster-size field",
			args: []string{"--custom-shard", "key-range=-80,cluster-size=PS_80,cluster-size=PS_20", "--custom-shard", "key-range=80-"},
			want: `duplicate --custom-shard field "cluster-size" in "key-range=-80,cluster-size=PS_80,cluster-size=PS_20"`,
		},
		{
			name: "duplicate key range",
			args: []string{"--custom-shard", "key-range=-80", "--custom-shard", "key-range=-80,cluster-size=PS_20"},
			want: `duplicate --custom-shard key-range "-80"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			var buf bytes.Buffer
			svc := &mock.KeyspacesService{
				CreateFn: func(_ context.Context, _ *ps.CreateKeyspaceRequest) (*ps.Keyspace, error) {
					c.Fatal("Create must not be called when flag validation fails")
					return nil, nil
				},
			}
			ch := newCreateTestHelper(c, &buf, svc)

			cmd := CreateCmd(ch)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(append([]string{"planetscale", "main", "events", "--cluster-size", "PS_40"}, tt.args...))
			c.Assert(cmd.Execute(), qt.ErrorMatches, tt.want)
			c.Assert(svc.CreateFnInvoked, qt.IsFalse)
		})
	}
}
