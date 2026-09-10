package sidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func sidecarTestHelper(svc ps.NekiSidecarsService, out *bytes.Buffer) *cmdutil.Helper {
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(out)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiSidecars: svc}, nil
		},
	}
}

func testSidecar() *ps.NekiSidecar {
	return &ps.NekiSidecar{
		Type:                 "neki_sidecar",
		ID:                   "sidecar-1",
		ConfigurationProfile: "metal",
		State:                "ready",
		CreatedAt:            time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		UpdatedAt:            time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
	}
}

func TestSidecarListCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiSidecarsService{
		ListFn: func(_ context.Context, req *ps.ListNekiSidecarsRequest) ([]*ps.NekiSidecar, error) {
			c.Assert(req, qt.DeepEquals, &ps.ListNekiSidecarsRequest{Organization: "acme", Database: "app", Branch: "main"})
			return []*ps.NekiSidecar{testSidecar()}, nil
		},
	}

	cmd := ListCmd(sidecarTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, []map[string]interface{}{{
		"type":                  "neki_sidecar",
		"id":                    "sidecar-1",
		"configuration_profile": "metal",
		"state":                 "ready",
		"created_at":            "2026-08-04T12:00:00Z",
		"updated_at":            "2026-08-04T12:00:00Z",
	}})
}

func TestSidecarListHumanHeadersAndUTC(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	sidecar := testSidecar()
	sidecar.CreatedAt = time.Date(2026, 8, 4, 12, 0, 0, 0, time.FixedZone("MDT", -6*60*60))
	svc := &mock.NekiSidecarsService{ListFn: func(context.Context, *ps.ListNekiSidecarsRequest) ([]*ps.NekiSidecar, error) {
		return []*ps.NekiSidecar{sidecar}, nil
	}}
	ch := &cmdutil.Helper{Printer: p, Config: &config.Config{Organization: "acme"}, Client: func() (*ps.Client, error) {
		return &ps.Client{NekiSidecars: svc}, nil
	}}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "ID")
	c.Assert(out.String(), qt.Contains, "CONFIGURATION PROFILE")
	c.Assert(out.String(), qt.Contains, "STATE")
	c.Assert(out.String(), qt.Contains, "CREATED AT")
	c.Assert(out.String(), qt.Contains, "sidecar-1")
	c.Assert(out.String(), qt.Contains, "metal")
	c.Assert(out.String(), qt.Contains, "2026-08-04 18:00:00")
}

func TestSidecarShowCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiSidecarsService{
		GetFn: func(_ context.Context, req *ps.GetNekiSidecarRequest) (*ps.NekiSidecar, error) {
			c.Assert(req, qt.DeepEquals, &ps.GetNekiSidecarRequest{
				Organization: "acme",
				Database:     "app",
				Branch:       "main",
				Sidecar:      "metal",
			})
			return testSidecar(), nil
		},
		ListParametersFn: func(_ context.Context, req *ps.ListNekiSidecarParametersRequest) ([]*ps.NekiParameter, error) {
			c.Assert(req.Sidecar, qt.Equals, "metal")
			return []*ps.NekiParameter{{
				Name:          "default_pool_size",
				Namespace:     "pgbouncer",
				ParameterType: "integer",
				DefaultValue:  "10",
				Value:         "20",
			}}, nil
		},
	}

	cmd := ShowCmd(sidecarTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.ListParametersFnInvoked, qt.IsTrue)
	var detail map[string]interface{}
	c.Assert(json.Unmarshal(out.Bytes(), &detail), qt.IsNil)
	c.Assert(detail["id"], qt.Equals, "sidecar-1")
	c.Assert(detail["configuration_profile"], qt.Equals, "metal")
	parameters := detail["parameters"].([]interface{})
	c.Assert(parameters, qt.HasLen, 1)
	c.Assert(parameters[0].(map[string]interface{})["name"], qt.Equals, "default_pool_size")
}

func TestSidecarUpdateCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiSidecarsService{UpdateFn: func(_ context.Context, req *ps.UpdateNekiSidecarRequest) (*ps.NekiSidecar, error) {
		c.Assert(req.Organization, qt.Equals, "acme")
		c.Assert(req.Database, qt.Equals, "app")
		c.Assert(req.Branch, qt.Equals, "main")
		c.Assert(req.Sidecar, qt.Equals, "metal")
		c.Assert(req.Parameters, qt.DeepEquals, map[string]map[string]string{
			"pgbouncer": {"default_pool_size": "20", "max_client_conn": "100"},
		})
		return testSidecar(), nil
	}}

	cmd := UpdateCmd(sidecarTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "--parameters", "pgbouncer.default_pool_size=20", "--parameters", "pgbouncer.max_client_conn=100"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestSidecarUpdateCmdRequiresParameters(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiSidecarsService{}

	cmd := UpdateCmd(sidecarTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, "at least one --parameters flag is required")
	c.Assert(svc.UpdateFnInvoked, qt.IsFalse)
}

func TestSidecarUpdateCmdInvalidParameters(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiSidecarsService{}

	cmd := UpdateCmd(sidecarTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "--parameters", "bogus"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, `invalid --parameters "bogus": expected namespace.name=value`)
}

func TestSidecarParametersCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiSidecarsService{ListParametersFn: func(_ context.Context, req *ps.ListNekiSidecarParametersRequest) ([]*ps.NekiParameter, error) {
		c.Assert(req, qt.DeepEquals, &ps.ListNekiSidecarParametersRequest{Organization: "acme", Database: "app", Branch: "main", Sidecar: "metal"})
		return []*ps.NekiParameter{
			{Name: "default_pool_size", Namespace: "pgbouncer", Value: "20"},
		}, nil
	}}

	cmd := ParametersCmd(sidecarTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, []map[string]interface{}{{
		"name": "default_pool_size", "display_name": "", "namespace": "pgbouncer", "advanced": false,
		"category": nil, "description": "", "parameter_type": "", "default_value": nil, "value": "20",
		"required": false, "created_at": "0001-01-01T00:00:00Z", "updated_at": nil, "restart": false, "url": "",
	}})
}

func TestSidecarChangesCancelCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiSidecarsService{CancelChangeFn: func(_ context.Context, req *ps.CancelNekiSidecarChangeRequest) error {
		c.Assert(req, qt.DeepEquals, &ps.CancelNekiSidecarChangeRequest{Organization: "acme", Database: "app", Branch: "main", Sidecar: "metal", Change: "change-1"})
		return nil
	}}

	cmd := changesCancelCmd(sidecarTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "change-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"result": "change canceled", "change_id": "change-1", "sidecar": "metal",
	})
}

func TestSidecarChangesListHumanDisplaysDifferences(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	svc := &mock.NekiSidecarsService{ListChangesFn: func(_ context.Context, req *ps.ListNekiSidecarChangesRequest) ([]*ps.NekiSidecarChange, error) {
		c.Assert(req, qt.DeepEquals, &ps.ListNekiSidecarChangesRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			Sidecar:      "metal",
			PerPage:      100,
		})
		return []*ps.NekiSidecarChange{{
			ID:        "change-1",
			State:     "completed",
			CreatedAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			Parameters: map[string]map[string]string{"pgbouncer": {
				"default_pool_size": "20",
				"max_client_conn":   "100",
			}},
			PreviousParameters: map[string]map[string]string{"pgbouncer": {
				"default_pool_size":  "10",
				"query_wait_timeout": "120",
			}},
		}}, nil
	}}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiSidecars: svc}, nil
		},
	}

	cmd := changesListCmd(ch)
	cmd.SetArgs([]string{"app", "main", "metal"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "CHANGES")
	c.Assert(out.String(), qt.Contains, "pgbouncer.default_pool_size: 10 → 20")
	c.Assert(out.String(), qt.Contains, "pgbouncer.max_client_conn: (default) → 100")
	c.Assert(out.String(), qt.Contains, "pgbouncer.query_wait_timeout: 120 → (default)")
}

func TestSidecarChangesShowHumanDisplaysDifferences(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	svc := &mock.NekiSidecarsService{GetChangeFn: func(_ context.Context, req *ps.GetNekiSidecarChangeRequest) (*ps.NekiSidecarChange, error) {
		c.Assert(req, qt.DeepEquals, &ps.GetNekiSidecarChangeRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			Sidecar:      "metal",
			Change:       "change-1",
		})
		return &ps.NekiSidecarChange{
			ID:        "change-1",
			State:     "applying",
			CreatedAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			Parameters: map[string]map[string]string{"pgbouncer": {
				"default_pool_size": "20",
				"max_client_conn":   "100",
			}},
			PreviousParameters: map[string]map[string]string{"pgbouncer": {
				"default_pool_size":  "10",
				"query_wait_timeout": "120",
			}},
		}, nil
	}}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiSidecars: svc}, nil
		},
	}

	cmd := changesShowCmd(ch)
	cmd.SetArgs([]string{"app", "main", "metal", "change-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "Changes:")
	c.Assert(out.String(), qt.Contains, "Parameter pgbouncer.default_pool_size: 10 → 20")
	c.Assert(out.String(), qt.Contains, "Parameter pgbouncer.max_client_conn: (default) → 100")
	c.Assert(out.String(), qt.Contains, "Parameter pgbouncer.query_wait_timeout: 120 → (default)")
}

func TestSidecarCmdSubcommands(t *testing.T) {
	cmd := SidecarCmd(&cmdutil.Helper{Config: &config.Config{}})
	for _, name := range []string{"changes", "list", "parameters", "show", "update"} {
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

	for _, sub := range cmd.Commands() {
		qt.Assert(t, sub.Name() == "create" || sub.Name() == "delete", qt.IsFalse)
	}
}
