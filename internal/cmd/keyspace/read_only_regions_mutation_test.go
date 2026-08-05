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

func readOnlyRegionsTestHelper(format printer.Format, svc *mock.KeyspacesService, out *bytes.Buffer) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(out)
	if format == printer.Human {
		p.SetHumanOutput(out)
	}
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}
}

func getReadOnlyRegionsFn(regions []*ps.ReadOnlyRegionKeyspace) func(context.Context, *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
	return func(_ context.Context, _ *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
		return &ps.Keyspace{ReadOnlyRegions: regions}, nil
	}
}

func TestKeyspace_ReadOnlyRegionsAddCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	current := []*ps.ReadOnlyRegionKeyspace{{
		Region:      "eu-west",
		ClusterName: "PS_10",
		Replicas:    1,
	}}
	result := append(current, &ps.ReadOnlyRegionKeyspace{
		Region:             "us-west",
		ClusterName:        "PS_20",
		ClusterDisplayName: "PS-20",
		Replicas:           2,
	})

	svc := &mock.KeyspacesService{
		GetFn: getReadOnlyRegionsFn(current),
		UpdateReadOnlyRegionsFn: func(_ context.Context, req *ps.UpdateReadOnlyRegionsRequest) ([]*ps.ReadOnlyRegionKeyspace, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "analytics")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.Keyspace, qt.Equals, "events")
			c.Assert(req.ReadOnlyRegions, qt.HasLen, 2)
			c.Assert(req.ReadOnlyRegions[0].Region, qt.Equals, "eu-west")
			c.Assert(*req.ReadOnlyRegions[0].ClusterSize, qt.Equals, "PS_10")
			c.Assert(*req.ReadOnlyRegions[0].Replicas, qt.Equals, 1)
			c.Assert(req.ReadOnlyRegions[1], qt.DeepEquals, &ps.ReadOnlyRegionKeyspaceConfig{Region: "us-west"})
			return result, nil
		},
	}

	cmd := ReadOnlyRegionsCmd(readOnlyRegionsTestHelper(printer.JSON, svc, &out))
	cmd.SetArgs([]string{"add", "analytics", "main", "events", "us-west"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, result)
}

func TestKeyspace_ReadOnlyRegionsAddCmdRejectsDuplicate(t *testing.T) {
	c := qt.New(t)
	current := []*ps.ReadOnlyRegionKeyspace{{Region: "us-west", ClusterName: "PS_10", Replicas: 1}}
	svc := &mock.KeyspacesService{
		GetFn: getReadOnlyRegionsFn(current),
		UpdateReadOnlyRegionsFn: func(context.Context, *ps.UpdateReadOnlyRegionsRequest) ([]*ps.ReadOnlyRegionKeyspace, error) {
			c.Fatal("UpdateReadOnlyRegions should not be called")
			return nil, nil
		},
	}

	cmd := ReadOnlyRegionsCmd(readOnlyRegionsTestHelper(printer.JSON, svc, &bytes.Buffer{}))
	cmd.SetArgs([]string{"add", "analytics", "main", "events", "us-west"})

	c.Assert(cmd.Execute(), qt.ErrorMatches, ".*already configured.*")
}

func TestKeyspace_ReadOnlyRegionsUpdateCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	current := []*ps.ReadOnlyRegionKeyspace{
		{Region: "us-west", ClusterName: "PS_10", Replicas: 2},
		{Region: "eu-west", ClusterName: "PS_20", Replicas: 1},
	}
	result := []*ps.ReadOnlyRegionKeyspace{
		{Region: "us-west", ClusterName: "PS_30", ClusterDisplayName: "PS-30", Replicas: 2},
		current[1],
	}
	svc := &mock.KeyspacesService{
		GetFn: getReadOnlyRegionsFn(current),
		UpdateReadOnlyRegionsFn: func(_ context.Context, req *ps.UpdateReadOnlyRegionsRequest) ([]*ps.ReadOnlyRegionKeyspace, error) {
			c.Assert(req.ReadOnlyRegions, qt.HasLen, 2)
			c.Assert(*req.ReadOnlyRegions[0].ClusterSize, qt.Equals, "PS_30")
			c.Assert(*req.ReadOnlyRegions[0].Replicas, qt.Equals, 2)
			c.Assert(*req.ReadOnlyRegions[1].ClusterSize, qt.Equals, "PS_20")
			c.Assert(*req.ReadOnlyRegions[1].Replicas, qt.Equals, 1)
			return result, nil
		},
	}

	cmd := ReadOnlyRegionsCmd(readOnlyRegionsTestHelper(printer.JSON, svc, &out))
	cmd.SetArgs([]string{"update", "analytics", "main", "events", "us-west", "--cluster-size", "PS_30"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, result)
}

func TestKeyspace_ReadOnlyRegionsUpdateCmdValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		regions []*ps.ReadOnlyRegionKeyspace
		want    string
	}{
		{
			name:    "requires sizing flag",
			args:    []string{"update", "analytics", "main", "events", "us-west"},
			regions: nil,
			want:    ".*at least one of --cluster-size or --replicas is required.*",
		},
		{
			name:    "requires configured region",
			args:    []string{"update", "analytics", "main", "events", "us-west", "--replicas", "2"},
			regions: []*ps.ReadOnlyRegionKeyspace{{Region: "eu-west", ClusterName: "PS_10", Replicas: 1}},
			want:    ".*is not configured.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			svc := &mock.KeyspacesService{
				GetFn: getReadOnlyRegionsFn(tt.regions),
				UpdateReadOnlyRegionsFn: func(context.Context, *ps.UpdateReadOnlyRegionsRequest) ([]*ps.ReadOnlyRegionKeyspace, error) {
					c.Fatal("UpdateReadOnlyRegions should not be called")
					return nil, nil
				},
			}
			cmd := ReadOnlyRegionsCmd(readOnlyRegionsTestHelper(printer.JSON, svc, &bytes.Buffer{}))
			cmd.SetArgs(tt.args)
			c.Assert(cmd.Execute(), qt.ErrorMatches, tt.want)
		})
	}
}

func TestKeyspace_ReadOnlyRegionsRemoveCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	current := []*ps.ReadOnlyRegionKeyspace{
		{Region: "us-west", ClusterName: "PS_10", Replicas: 2},
		{Region: "eu-west", ClusterName: "PS_20", Replicas: 1},
	}
	svc := &mock.KeyspacesService{
		GetFn: getReadOnlyRegionsFn(current),
		UpdateReadOnlyRegionsFn: func(_ context.Context, req *ps.UpdateReadOnlyRegionsRequest) ([]*ps.ReadOnlyRegionKeyspace, error) {
			c.Assert(req.ReadOnlyRegions, qt.HasLen, 1)
			c.Assert(req.ReadOnlyRegions[0].Region, qt.Equals, "eu-west")
			c.Assert(*req.ReadOnlyRegions[0].ClusterSize, qt.Equals, "PS_20")
			c.Assert(*req.ReadOnlyRegions[0].Replicas, qt.Equals, 1)
			return current[1:], nil
		},
	}

	cmd := ReadOnlyRegionsCmd(readOnlyRegionsTestHelper(printer.Human, svc, &out))
	cmd.SetArgs([]string{"remove", "analytics", "main", "events", "us-west"})

	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "Removed read-only region us-west from keyspace events.")
}

func TestKeyspace_ReadOnlyRegionsRemoveCmdRejectsMissingRegion(t *testing.T) {
	c := qt.New(t)
	svc := &mock.KeyspacesService{
		GetFn: getReadOnlyRegionsFn(nil),
		UpdateReadOnlyRegionsFn: func(context.Context, *ps.UpdateReadOnlyRegionsRequest) ([]*ps.ReadOnlyRegionKeyspace, error) {
			c.Fatal("UpdateReadOnlyRegions should not be called")
			return nil, nil
		},
	}

	cmd := ReadOnlyRegionsCmd(readOnlyRegionsTestHelper(printer.JSON, svc, &bytes.Buffer{}))
	cmd.SetArgs([]string{"remove", "analytics", "main", "events", "us-west"})

	c.Assert(cmd.Execute(), qt.ErrorMatches, ".*is not configured.*")
}
