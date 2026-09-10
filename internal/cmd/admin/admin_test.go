package admin

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

func adminTestHelper(svc ps.NekiAdminsService, out *bytes.Buffer) *cmdutil.Helper {
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(out)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiAdmins: svc}, nil
		},
	}
}

func testAdmin() *ps.NekiAdmin {
	return &ps.NekiAdmin{
		Type:      "NekiAdmin",
		ID:        "admin-1",
		SKU:       &ps.NekiAdminSKU{Name: "NKA_0", DisplayName: "NKA-0"},
		AdminSize: "NKA_0",
		State:     "ready",
		CreatedAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
	}
}

func TestAdminShowCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiAdminsService{
		GetFn: func(_ context.Context, req *ps.GetNekiAdminRequest) (*ps.NekiAdmin, error) {
			c.Assert(req, qt.DeepEquals, &ps.GetNekiAdminRequest{
				Organization: "acme",
				Database:     "app",
				Branch:       "main",
			})
			return testAdmin(), nil
		},
		ListParametersFn: func(_ context.Context, req *ps.ListNekiAdminParametersRequest) ([]*ps.NekiParameter, error) {
			c.Assert(req, qt.DeepEquals, &ps.ListNekiAdminParametersRequest{
				Organization: "acme",
				Database:     "app",
				Branch:       "main",
			})
			return []*ps.NekiParameter{{
				Name:          "recovery-poll-interval",
				Namespace:     "admin",
				ParameterType: "time",
				DefaultValue:  "10s",
				Value:         "20s",
			}}, nil
		},
	}

	cmd := ShowCmd(adminTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.ListParametersFnInvoked, qt.IsTrue)
	var detail map[string]interface{}
	c.Assert(json.Unmarshal(out.Bytes(), &detail), qt.IsNil)
	c.Assert(detail["id"], qt.Equals, "admin-1")
	c.Assert(detail["admin_size"], qt.Equals, "NKA_0")
	parameters := detail["parameters"].([]interface{})
	c.Assert(parameters, qt.HasLen, 1)
	c.Assert(parameters[0].(map[string]interface{})["name"], qt.Equals, "recovery-poll-interval")
}

func TestAdminShowHumanHeadersAndUTC(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	admin := testAdmin()
	admin.CreatedAt = time.Date(2026, 8, 4, 12, 0, 0, 0, time.FixedZone("MDT", -6*60*60))
	svc := &mock.NekiAdminsService{
		GetFn: func(context.Context, *ps.GetNekiAdminRequest) (*ps.NekiAdmin, error) {
			return admin, nil
		},
		ListParametersFn: func(context.Context, *ps.ListNekiAdminParametersRequest) ([]*ps.NekiParameter, error) {
			return nil, nil
		},
	}
	ch := &cmdutil.Helper{Printer: p, Config: &config.Config{Organization: "acme"}, Client: func() (*ps.Client, error) {
		return &ps.Client{NekiAdmins: svc}, nil
	}}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "ID")
	c.Assert(out.String(), qt.Contains, "SIZE")
	c.Assert(out.String(), qt.Contains, "STATE")
	c.Assert(out.String(), qt.Contains, "CREATED AT")
	c.Assert(out.String(), qt.Contains, "admin-1")
	c.Assert(out.String(), qt.Contains, "NKA-0")
	c.Assert(out.String(), qt.Contains, "2026-08-04 18:00:00")
}

func TestAdminUpdateCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiAdminsService{UpdateFn: func(_ context.Context, req *ps.UpdateNekiAdminRequest) (*ps.NekiAdmin, error) {
		c.Assert(req.Organization, qt.Equals, "acme")
		c.Assert(req.Database, qt.Equals, "app")
		c.Assert(req.Branch, qt.Equals, "main")
		c.Assert(*req.AdminSize, qt.Equals, "NKA_1")
		c.Assert(req.Parameters, qt.DeepEquals, map[string]map[string]string{
			"admin": {"recovery-poll-interval": "20s"},
		})
		return testAdmin(), nil
	}}

	cmd := UpdateCmd(adminTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "--size", "NKA-1", "--parameters", "admin.recovery-poll-interval=20s"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestAdminUpdateCmdRequiresFlag(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiAdminsService{}

	cmd := UpdateCmd(adminTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, "at least one of --size or --parameters is required")
	c.Assert(svc.UpdateFnInvoked, qt.IsFalse)
}

func TestAdminUpdateCmdInvalidParameters(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiAdminsService{}

	cmd := UpdateCmd(adminTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "--parameters", "bogus"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, `invalid --parameters "bogus": expected namespace.name=value`)
}

func TestAdminParametersCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiAdminsService{ListParametersFn: func(_ context.Context, req *ps.ListNekiAdminParametersRequest) ([]*ps.NekiParameter, error) {
		c.Assert(req, qt.DeepEquals, &ps.ListNekiAdminParametersRequest{Organization: "acme", Database: "app", Branch: "main"})
		return []*ps.NekiParameter{
			{Name: "recovery-poll-interval", Namespace: "admin", Value: "20s"},
		}, nil
	}}

	cmd := ParametersCmd(adminTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, []map[string]interface{}{{
		"name": "recovery-poll-interval", "display_name": "", "namespace": "admin", "advanced": false,
		"category": nil, "description": "", "parameter_type": "", "default_value": nil, "value": "20s",
		"required": false, "created_at": "0001-01-01T00:00:00Z", "updated_at": nil, "restart": false, "url": "",
	}})
}

func TestAdminChangesCancelCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiAdminsService{CancelChangeFn: func(_ context.Context, req *ps.CancelNekiAdminChangeRequest) error {
		c.Assert(req, qt.DeepEquals, &ps.CancelNekiAdminChangeRequest{Organization: "acme", Database: "app", Branch: "main", Change: "change-1"})
		return nil
	}}

	cmd := changesCancelCmd(adminTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "change-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"result": "change canceled", "change_id": "change-1",
	})
}

func TestAdminChangesListHumanDisplaysDifferences(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	svc := &mock.NekiAdminsService{ListChangesFn: func(_ context.Context, req *ps.ListNekiAdminChangesRequest) ([]*ps.NekiAdminChange, error) {
		c.Assert(req, qt.DeepEquals, &ps.ListNekiAdminChangesRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			PerPage:      100,
		})
		return []*ps.NekiAdminChange{{
			ID:                           "change-1",
			State:                        "completed",
			CreatedAt:                    time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			AdminSize:                    "NKA_1",
			AdminSizeDisplayName:         "NKA-1",
			PreviousAdminSize:            "NKA_0",
			PreviousAdminSizeDisplayName: "NKA-0",
			Parameters: map[string]map[string]string{"admin": {
				"recovery-poll-interval": "20s",
			}},
			PreviousParameters: map[string]map[string]string{"admin": {
				"recovery-poll-interval":     "10s",
				"planned-switchover-timeout": "1m",
			}},
		}}, nil
	}}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiAdmins: svc}, nil
		},
	}

	cmd := changesListCmd(ch)
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "CHANGES")
	c.Assert(out.String(), qt.Contains, "NKA-0")
	c.Assert(out.String(), qt.Contains, "NKA-1")
	c.Assert(out.String(), qt.Contains, "admin.recovery-poll-interval")
	c.Assert(out.String(), qt.Contains, "admin.planned-switchover-timeout")
}

func TestFormatChangeSummary(t *testing.T) {
	c := qt.New(t)
	got := formatChangeSummary(&ps.NekiAdminChange{
		AdminSize:                    "NKA_1",
		AdminSizeDisplayName:         "NKA-1",
		PreviousAdminSize:            "NKA_0",
		PreviousAdminSizeDisplayName: "NKA-0",
		Parameters: map[string]map[string]string{"admin": {
			"recovery-poll-interval": "20s",
		}},
		PreviousParameters: map[string]map[string]string{"admin": {
			"recovery-poll-interval":     "10s",
			"planned-switchover-timeout": "1m",
		}},
	})
	c.Assert(got, qt.Contains, "Size: NKA-0 → NKA-1")
	c.Assert(got, qt.Contains, "Parameter admin.recovery-poll-interval: 10s → 20s")
	c.Assert(got, qt.Contains, "Parameter admin.planned-switchover-timeout: 1m → (default)")
}

func TestAdminChangesShowHumanDisplaysDifferences(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	svc := &mock.NekiAdminsService{GetChangeFn: func(_ context.Context, req *ps.GetNekiAdminChangeRequest) (*ps.NekiAdminChange, error) {
		c.Assert(req, qt.DeepEquals, &ps.GetNekiAdminChangeRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			Change:       "change-1",
		})
		return &ps.NekiAdminChange{
			ID:                           "change-1",
			State:                        "applying",
			CreatedAt:                    time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			AdminSize:                    "NKA_1",
			AdminSizeDisplayName:         "NKA-1",
			PreviousAdminSize:            "NKA_0",
			PreviousAdminSizeDisplayName: "NKA-0",
			Parameters: map[string]map[string]string{"admin": {
				"recovery-poll-interval": "20s",
			}},
			PreviousParameters: map[string]map[string]string{"admin": {
				"recovery-poll-interval":     "10s",
				"planned-switchover-timeout": "1m",
			}},
		}, nil
	}}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiAdmins: svc}, nil
		},
	}

	cmd := changesShowCmd(ch)
	cmd.SetArgs([]string{"app", "main", "change-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "Changes:")
	c.Assert(out.String(), qt.Contains, "Size: NKA-0 → NKA-1")
	c.Assert(out.String(), qt.Contains, "Parameter admin.recovery-poll-interval: 10s → 20s")
	c.Assert(out.String(), qt.Contains, "Parameter admin.planned-switchover-timeout: 1m → (default)")
}

func TestAdminSizesCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiAdminsService{ListSizeSKUsFn: func(_ context.Context, req *ps.ListNekiAdminSizeSKUsRequest) ([]*ps.NekiAdminSKU, error) {
		c.Assert(req, qt.DeepEquals, &ps.ListNekiAdminSizeSKUsRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
		})
		return []*ps.NekiAdminSKU{{
			Name:        "NKA_0",
			DisplayName: "NKA-0",
			CPU:         "0.5",
			RAM:         1073741824,
			SortOrder:   1,
		}}, nil
	}}

	cmd := SizesCmd(adminTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListSizeSKUsFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, []map[string]interface{}{{
		"name":         "NKA_0",
		"display_name": "NKA-0",
		"cpu":          "0.5",
		"ram":          float64(1073741824),
		"sort_order":   float64(1),
	}})
}

func TestAdminCmdSubcommands(t *testing.T) {
	cmd := AdminCmd(&cmdutil.Helper{Config: &config.Config{}})
	for _, name := range []string{"changes", "parameters", "show", "sizes", "update"} {
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
		qt.Assert(t, sub.Name() == "create" || sub.Name() == "delete" || sub.Name() == "list", qt.IsFalse)
	}
}
