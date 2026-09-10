package router

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func routerTestHelper(svc ps.NekiRoutersService, out *bytes.Buffer) *cmdutil.Helper {
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(out)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiRouters: svc}, nil
		},
	}
}

func testRouter() *ps.NekiRouter {
	return &ps.NekiRouter{
		Type:            "neki_router",
		Name:            "default",
		Default:         true,
		SKU:             &ps.NekiRouterSKU{Name: "NKR_5", DisplayName: "NKR-5", CPU: "1", RAM: 4294967296, SortOrder: 1},
		ReplicasPerCell: 1,
		State:           "ready",
		CreatedAt:       time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
	}
}

func TestRouterListCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiRoutersService{
		ListFn: func(_ context.Context, req *ps.ListNekiRoutersRequest) ([]*ps.NekiRouter, error) {
			c.Assert(req, qt.DeepEquals, &ps.ListNekiRoutersRequest{
				Organization: "acme",
				Database:     "app",
				Branch:       "main",
			})
			return []*ps.NekiRouter{testRouter()}, nil
		},
	}

	cmd := ListCmd(routerTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, []map[string]interface{}{{
		"type":                   "neki_router",
		"name":                   "default",
		"default":                true,
		"sku":                    map[string]interface{}{"name": "NKR_5", "display_name": "NKR-5", "cpu": "1", "ram": float64(4294967296), "sort_order": float64(1)},
		"replicas_per_cell":      float64(1),
		"autoscaling":            false,
		"max_replicas_per_cell":  nil,
		"target_cpu_utilization": nil,
		"state":                  "ready",
		"created_at":             "2026-08-04T12:00:00Z",
		"updated_at":             "2026-08-04T12:00:00Z",
	}})
}

func TestRouterListHumanHeaders(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)

	svc := &mock.NekiRoutersService{
		ListFn: func(context.Context, *ps.ListNekiRoutersRequest) ([]*ps.NekiRouter, error) {
			return []*ps.NekiRouter{testRouter()}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiRouters: svc}, nil
		},
	}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "NAME")
	c.Assert(out.String(), qt.Contains, "SIZE")
	c.Assert(out.String(), qt.Contains, "REPLICAS PER CELL")
	c.Assert(out.String(), qt.Contains, "AUTOSCALING")
	c.Assert(out.String(), qt.Contains, "STATE")
	c.Assert(out.String(), qt.Contains, "default")
	c.Assert(out.String(), qt.Contains, "NKR-5")
	c.Assert(out.String(), qt.Contains, "ready")
}

func TestRouterShowCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiRoutersService{
		GetFn: func(_ context.Context, req *ps.GetNekiRouterRequest) (*ps.NekiRouter, error) {
			c.Assert(req, qt.DeepEquals, &ps.GetNekiRouterRequest{
				Organization: "acme",
				Database:     "app",
				Branch:       "main",
				Router:       "default",
			})
			return testRouter(), nil
		},
		ListParametersFn: func(_ context.Context, req *ps.ListNekiRouterParametersRequest) ([]*ps.NekiParameter, error) {
			c.Assert(req.Router, qt.Equals, "default")
			return []*ps.NekiParameter{{
				Name:          "replication-lag-tolerable-max",
				Namespace:     "router",
				ParameterType: "duration",
				DefaultValue:  "10m",
				Value:         "15m",
			}}, nil
		},
	}

	cmd := ShowCmd(routerTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "default"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.ListParametersFnInvoked, qt.IsTrue)
	var detail map[string]interface{}
	c.Assert(json.Unmarshal(out.Bytes(), &detail), qt.IsNil)
	c.Assert(detail["name"], qt.Equals, "default")
	parameters := detail["parameters"].([]interface{})
	c.Assert(parameters, qt.HasLen, 1)
	c.Assert(parameters[0].(map[string]interface{})["name"], qt.Equals, "replication-lag-tolerable-max")
}

func TestRouterCreateCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiRoutersService{
		CreateFn: func(_ context.Context, req *ps.CreateNekiRouterRequest) (*ps.NekiRouter, error) {
			c.Assert(req, qt.DeepEquals, &ps.CreateNekiRouterRequest{
				Organization:    "acme",
				Database:        "app",
				Branch:          "main",
				Name:            "analytics",
				RouterSize:      "NKR_10",
				ReplicasPerCell: &[]int{2}[0],
			})
			return &ps.NekiRouter{Name: "analytics", State: "provisioning", ReplicasPerCell: 2}, nil
		},
	}

	cmd := CreateCmd(routerTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "analytics", "--size", "NKR-10", "--replicas-per-cell", "2"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	var created map[string]interface{}
	c.Assert(json.Unmarshal(out.Bytes(), &created), qt.IsNil)
	c.Assert(created["name"], qt.Equals, "analytics")
	c.Assert(created["state"], qt.Equals, "provisioning")
}

func TestRouterCreateCmdDefaults(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiRoutersService{
		CreateFn: func(_ context.Context, req *ps.CreateNekiRouterRequest) (*ps.NekiRouter, error) {
			c.Assert(req.RouterSize, qt.Equals, "")
			c.Assert(req.ReplicasPerCell, qt.IsNil)
			return &ps.NekiRouter{Name: req.Name, State: "provisioning"}, nil
		},
	}

	cmd := CreateCmd(routerTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "analytics"})
	c.Assert(cmd.Execute(), qt.IsNil)
}

func TestRouterUpdateCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiRoutersService{
		UpdateFn: func(_ context.Context, req *ps.UpdateNekiRouterRequest) (*ps.NekiRouter, error) {
			c.Assert(req.Organization, qt.Equals, "acme")
			c.Assert(req.Router, qt.Equals, "analytics")
			c.Assert(req.RouterSize, qt.IsNil)
			c.Assert(req.ReplicasPerCell, qt.IsNil)
			c.Assert(*req.Autoscaling, qt.IsTrue)
			c.Assert(*req.MaxReplicasPerCell, qt.Equals, 4)
			c.Assert(*req.TargetCPUUtilization, qt.Equals, 70)
			c.Assert(req.Parameters, qt.DeepEquals, map[string]map[string]string{
				"router": {"replication-lag-tolerable-max": "15m"},
			})
			return &ps.NekiRouter{Name: "analytics", State: "updating", Autoscaling: true}, nil
		},
	}

	cmd := UpdateCmd(routerTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "analytics",
		"--autoscaling", "--max-replicas-per-cell", "4", "--target-cpu-utilization", "70",
		"--parameters", "router.replication-lag-tolerable-max=15m",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestRouterUpdateCmdRequiresFlag(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiRoutersService{}

	cmd := UpdateCmd(routerTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "analytics"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, "at least one update flag is required")
	c.Assert(svc.UpdateFnInvoked, qt.IsFalse)
}

func TestRouterUpdateCmdInvalidParameters(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiRoutersService{}

	cmd := UpdateCmd(routerTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "analytics", "--parameters", "bogus"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, `invalid --parameters "bogus": expected namespace.name=value`)
}

func TestRouterDeleteCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiRoutersService{
		DeleteFn: func(_ context.Context, req *ps.DeleteNekiRouterRequest) error {
			c.Assert(req, qt.DeepEquals, &ps.DeleteNekiRouterRequest{
				Organization: "acme",
				Database:     "app",
				Branch:       "main",
				Router:       "analytics",
			})
			return nil
		},
	}

	cmd := DeleteCmd(routerTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "analytics", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"result": "router deleted",
		"router": "analytics",
		"branch": "main",
	})
}

func TestRouterChangesCancelCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiRoutersService{CancelChangeFn: func(_ context.Context, req *ps.CancelNekiRouterChangeRequest) error {
		c.Assert(req, qt.DeepEquals, &ps.CancelNekiRouterChangeRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			Router:       "analytics",
			Change:       "change-1",
		})
		return nil
	}}

	cmd := changesCancelCmd(routerTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "analytics", "change-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"result": "change canceled", "change_id": "change-1", "router": "analytics",
	})
}

func TestRouterChangesListHumanDisplaysDifferences(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	autoscaling := true
	maxReplicas := 4
	targetCPU := 70
	svc := &mock.NekiRoutersService{ListChangesFn: func(_ context.Context, req *ps.ListNekiRouterChangesRequest) ([]*ps.NekiRouterChange, error) {
		c.Assert(req, qt.DeepEquals, &ps.ListNekiRouterChangesRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			Router:       "analytics",
			PerPage:      100,
		})
		return []*ps.NekiRouterChange{{
			ID:                            "change-1",
			State:                         "completed",
			CreatedAt:                     time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			RouterSize:                    "NKR_10",
			RouterSizeDisplayName:         "NKR-10",
			PreviousRouterSize:            "NKR_5",
			PreviousRouterSizeDisplayName: "NKR-5",
			ReplicasPerCell:               2,
			PreviousReplicasPerCell:       1,
			Autoscaling:                   &autoscaling,
			MaxReplicasPerCell:            &maxReplicas,
			TargetCPUUtilization:          &targetCPU,
			Parameters:                    map[string]map[string]string{"router": {"replication-lag-tolerable-max": "15m"}},
			PreviousParameters:            map[string]map[string]string{"router": {"replication-lag-tolerable-max": "10m"}},
		}}, nil
	}}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiRouters: svc}, nil
		},
	}

	cmd := changesListCmd(ch)
	cmd.SetArgs([]string{"app", "main", "analytics"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "CHANGES")
	c.Assert(out.String(), qt.Contains, "Size: NKR-5 → NKR-10")
	c.Assert(out.String(), qt.Contains, "Replicas per cell: 1 → 2")
	c.Assert(out.String(), qt.Contains, "Autoscaling: (default) → true")
	c.Assert(out.String(), qt.Contains, "Max replicas per")
	c.Assert(out.String(), qt.Contains, "(default) → 4")
	c.Assert(out.String(), qt.Contains, "Target CPU utilization")
	c.Assert(out.String(), qt.Contains, "(default) → 70")
	c.Assert(out.String(), qt.Contains, "router.replication-lag-tolerable-max: 10m → 15m")
}

func TestRouterChangesShowHumanDisplaysDifferences(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	svc := &mock.NekiRoutersService{GetChangeFn: func(_ context.Context, req *ps.GetNekiRouterChangeRequest) (*ps.NekiRouterChange, error) {
		c.Assert(req, qt.DeepEquals, &ps.GetNekiRouterChangeRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			Router:       "analytics",
			Change:       "change-1",
		})
		return &ps.NekiRouterChange{
			ID:                      "change-1",
			State:                   "applying",
			CreatedAt:               time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			RouterSize:              "NKR_10",
			PreviousRouterSize:      "NKR_5",
			ReplicasPerCell:         2,
			PreviousReplicasPerCell: 1,
			Parameters:              map[string]map[string]string{"router": {"replication-lag-tolerable-max": "15m"}},
			PreviousParameters:      map[string]map[string]string{"router": {"replication-lag-tolerable-max": "10m"}},
		}, nil
	}}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiRouters: svc}, nil
		},
	}

	cmd := changesShowCmd(ch)
	cmd.SetArgs([]string{"app", "main", "analytics", "change-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "Changes:")
	c.Assert(out.String(), qt.Contains, "Size: NKR_5 → NKR_10")
	c.Assert(out.String(), qt.Contains, "Replicas per cell: 1 → 2")
	c.Assert(out.String(), qt.Contains, "Parameter router.replication-lag-tolerable-max: 10m → 15m")
}

func TestRouterSizesCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	rate := int64(25)
	svc := &mock.NekiRoutersService{ListSizeSKUsFn: func(_ context.Context, req *ps.ListNekiRouterSizeSKUsRequest) ([]*ps.NekiRouterSKU, error) {
		c.Assert(req, qt.DeepEquals, &ps.ListNekiRouterSizeSKUsRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			Rates:        true,
		})
		return []*ps.NekiRouterSKU{{
			Name:        "NKR_5",
			DisplayName: "NKR-5",
			CPU:         "1",
			RAM:         4294967296,
			SortOrder:   1,
			Rate:        &rate,
		}}, nil
	}}

	cmd := SizesCmd(routerTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListSizeSKUsFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, []map[string]interface{}{{
		"name":         "NKR_5",
		"display_name": "NKR-5",
		"cpu":          "1",
		"ram":          float64(4294967296),
		"sort_order":   float64(1),
		"rate":         float64(25),
	}})
}

func TestRouterSizesHumanHeaders(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	rate := int64(25)
	svc := &mock.NekiRoutersService{ListSizeSKUsFn: func(context.Context, *ps.ListNekiRouterSizeSKUsRequest) ([]*ps.NekiRouterSKU, error) {
		return []*ps.NekiRouterSKU{{Name: "NKR_5", DisplayName: "NKR-5", CPU: "1", RAM: 4294967296, Rate: &rate}}, nil
	}}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiRouters: svc}, nil
		},
	}

	cmd := SizesCmd(ch)
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "NAME")
	c.Assert(out.String(), qt.Contains, "NKR-5")
	c.Assert(out.String(), qt.Contains, "$25")
}

func TestRouterCmdSubcommands(t *testing.T) {
	cmd := RouterCmd(&cmdutil.Helper{Config: &config.Config{}})
	for _, name := range []string{"changes", "create", "delete", "list", "show", "sizes", "update"} {
		found, _, err := cmd.Find([]string{name})
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, found.Name(), qt.Equals, name)
	}

	changes, _, err := cmd.Find([]string{"changes"})
	qt.Assert(t, err, qt.IsNil)
	for _, name := range []string{"cancel", "list", "show"} {
		found, _, err := changes.Find([]string{name})
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, found.Name(), qt.Equals, name)
	}
}
