package shard

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func shardTestHelper(svc ps.NekiShardsService, out *bytes.Buffer) *cmdutil.Helper {
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(out)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiShards: svc}, nil
		},
	}
}

func TestShardListCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	displayName := "reports"
	svc := &mock.NekiShardsService{
		ListFn: func(_ context.Context, req *ps.ListNekiShardsRequest) ([]*ps.NekiShard, error) {
			c.Assert(req, qt.DeepEquals, &ps.ListNekiShardsRequest{
				Organization:         "acme",
				Database:             "app",
				Branch:               "main",
				ConfigurationProfile: "analytics",
				Query:                "report",
				Page:                 2,
				PerPage:              20,
			})
			return []*ps.NekiShard{{
				ID:                   "shard-1",
				Name:                 "0",
				DisplayName:          &displayName,
				ConfigurationProfile: "analytics",
				Ready:                true,
				Authoritative:        true,
				CreatedAt:            time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
	}

	cmd := ListCmd(shardTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "--config-profile", "analytics", "--query", "report", "--page", "2", "--per-page", "20"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, []map[string]interface{}{{
		"id":                    "shard-1",
		"name":                  "0",
		"display_name":          "reports",
		"configuration_profile": "analytics",
		"ready":                 true,
		"authoritative":         true,
		"created_at":            "2026-08-04T12:00:00Z",
	}})
}

func TestShardListHumanHeaders(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)

	svc := &mock.NekiShardsService{
		ListFn: func(context.Context, *ps.ListNekiShardsRequest) ([]*ps.NekiShard, error) {
			return []*ps.NekiShard{{
				ID:                   "shard-1",
				Name:                 "0",
				ConfigurationProfile: "default",
				Ready:                true,
				Authoritative:        true,
				CreatedAt:            time.Date(2026, 8, 4, 12, 0, 0, 0, time.FixedZone("MDT", -6*60*60)),
			}}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiShards: svc}, nil
		},
	}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "DISPLAY NAME")
	c.Assert(out.String(), qt.Contains, "CONFIGURATION PROFILE")
	c.Assert(out.String(), qt.Contains, "READY")
	c.Assert(out.String(), qt.Contains, "AUTHORITATIVE")
	c.Assert(out.String(), qt.Contains, "CREATED AT")
	c.Assert(out.String(), qt.Contains, "Yes")
	c.Assert(out.String(), qt.Contains, "2026-08-04 18:00:00")
}

func TestShardCreateCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardsService{
		CreateFn: func(_ context.Context, req *ps.CreateNekiShardsRequest) (*ps.CreateNekiShardsResponse, error) {
			c.Assert(req, qt.DeepEquals, &ps.CreateNekiShardsRequest{
				Organization:         "acme",
				Database:             "app",
				Branch:               "main",
				ConfigurationProfile: "analytics",
				Count:                2,
			})
			return &ps.CreateNekiShardsResponse{Created: 2, ShardIDs: []string{"shard-1", "shard-2"}}, nil
		},
	}

	cmd := CreateCmd(shardTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "--config-profile", "analytics", "--count", "2"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"created":   2,
		"shard_ids": []interface{}{"shard-1", "shard-2"},
	})
}

func TestShardAssignCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardsService{
		AssignFn: func(_ context.Context, req *ps.AssignNekiShardsRequest) ([]*ps.NekiShardAssignment, error) {
			c.Assert(req, qt.DeepEquals, &ps.AssignNekiShardsRequest{
				Organization:         "acme",
				Database:             "app",
				Branch:               "main",
				ConfigurationProfile: "analytics",
				ShardIDs:             []string{"shard-1", "shard-2"},
			})
			return []*ps.NekiShardAssignment{
				{ID: "shard-1", Status: "assigned"},
				{ID: "shard-2", Status: "failed", Error: map[string]interface{}{"code": "incompatible_architecture"}},
			}, nil
		},
	}

	cmd := AssignCmd(shardTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "shard-1", "shard-2", "--config-profile", "analytics"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, []map[string]interface{}{
		{"id": "shard-1", "status": "assigned"},
		{"id": "shard-2", "status": "failed", "error": map[string]interface{}{"code": "incompatible_architecture"}},
	})
}

func TestShardUpdateCmdCanClearDisplayName(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardsService{
		UpdateFn: func(_ context.Context, req *ps.UpdateNekiShardRequest) (*ps.NekiShard, error) {
			c.Assert(req.DisplayName, qt.IsNil)
			return &ps.NekiShard{ID: req.Shard, Name: "0", DisplayName: req.DisplayName, ConfigurationProfile: "default"}, nil
		},
	}

	cmd := UpdateCmd(shardTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "shard-1", "--display-name="})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"id":                    "shard-1",
		"name":                  "0",
		"display_name":          nil,
		"configuration_profile": "default",
		"ready":                 false,
		"authoritative":         false,
		"created_at":            "0001-01-01T00:00:00Z",
	})
}

func TestShardUpdateCmdPreservesDisplayName(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardsService{
		UpdateFn: func(_ context.Context, req *ps.UpdateNekiShardRequest) (*ps.NekiShard, error) {
			c.Assert(req.DisplayName, qt.IsNotNil)
			c.Assert(*req.DisplayName, qt.Equals, "reporting")
			return &ps.NekiShard{ID: req.Shard, Name: "0", DisplayName: req.DisplayName, ConfigurationProfile: "default"}, nil
		},
	}

	cmd := UpdateCmd(shardTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "shard-1", "--display-name=reporting"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"id":                    "shard-1",
		"name":                  "0",
		"display_name":          "reporting",
		"configuration_profile": "default",
		"ready":                 false,
		"authoritative":         false,
		"created_at":            "0001-01-01T00:00:00Z",
	})
}

func TestShardShowCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	displayName := "reports"
	svc := &mock.NekiShardsService{
		GetFn: func(_ context.Context, req *ps.GetNekiShardRequest) (*ps.NekiShard, error) {
			c.Assert(req, qt.DeepEquals, &ps.GetNekiShardRequest{
				Organization: "acme",
				Database:     "app",
				Branch:       "main",
				Shard:        "shard-1",
			})
			return &ps.NekiShard{
				ID:                   "shard-1",
				Name:                 "0",
				DisplayName:          &displayName,
				ConfigurationProfile: "analytics",
				Ready:                true,
				Authoritative:        true,
				CreatedAt:            time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			}, nil
		},
	}

	cmd := ShowCmd(shardTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "shard-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"id":                    "shard-1",
		"name":                  "0",
		"display_name":          "reports",
		"configuration_profile": "analytics",
		"ready":                 true,
		"authoritative":         true,
		"created_at":            "2026-08-04T12:00:00Z",
	})
}

func TestShardDeleteCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardsService{
		DeleteFn: func(_ context.Context, req *ps.DeleteNekiShardRequest) error {
			c.Assert(req, qt.DeepEquals, &ps.DeleteNekiShardRequest{
				Organization: "acme",
				Database:     "app",
				Branch:       "main",
				Shard:        "shard-1",
			})
			return nil
		},
	}

	cmd := DeleteCmd(shardTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "shard-1", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"result":   "shard deleted",
		"shard_id": "shard-1",
		"branch":   "main",
	})
}

func TestShardCmdSubcommands(t *testing.T) {
	cmd := ShardCmd(&cmdutil.Helper{Config: &config.Config{}})
	for _, name := range []string{"assign", "create", "delete", "list", "show", "update"} {
		found, _, err := cmd.Find([]string{name})
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, found.Name(), qt.Equals, name)
	}
}
