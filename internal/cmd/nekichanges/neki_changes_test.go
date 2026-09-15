package nekichanges

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

func changesTestHelper(svc ps.NekiChangesService, out *bytes.Buffer) *cmdutil.Helper {
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(out)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiChanges: svc}, nil
		},
	}
}

func testChange() *ps.NekiChange {
	change := &ps.NekiChange{}
	err := json.Unmarshal([]byte(`{
		"type": "NekiAdminChangeRequest",
		"id": "change-1",
		"state": "pending",
		"target_id": "admin-1",
		"target_type": "NekiAdmin",
		"target_name": "Admin",
		"created_at": "2026-08-04T12:00:00Z",
		"updated_at": "2026-08-04T12:00:00Z",
		"admin_size": "NKA_1"
	}`), change)
	if err != nil {
		panic(err)
	}
	return change
}

func TestChangesCmdSubcommands(t *testing.T) {
	cmd := ChangesCmd(&cmdutil.Helper{Config: &config.Config{}})
	qt.Assert(t, cmd.Name(), qt.Equals, "changes")
	qt.Assert(t, cmd.Aliases, qt.DeepEquals, []string{"neki-changes"})
	for _, name := range []string{"cancel", "list", "show"} {
		found, _, err := cmd.Find([]string{name})
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, found.Name(), qt.Equals, name)
	}
}

func TestChangesListCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiChangesService{ListFn: func(_ context.Context, req *ps.ListNekiChangesRequest) ([]*ps.NekiChange, error) {
		c.Assert(req, qt.DeepEquals, &ps.ListNekiChangesRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			States:       []string{"pending", "applying"},
			TargetTypes:  []string{"NekiAdmin", "NekiConfigurationProfile"},
			TargetID:     "admin-1",
			Period:       "24h",
			CompletedAt:  "2026-08-04",
			Page:         2,
			PerPage:      25,
		})
		return []*ps.NekiChange{testChange()}, nil
	}}

	cmd := changesListCmd(changesTestHelper(svc, &out))
	cmd.SetArgs([]string{
		"app", "main",
		"--state", "pending,applying",
		"--target-type", "admin",
		"--target-type", "config-profile",
		"--target-id", "admin-1",
		"--period", "24h",
		"--completed-at", "2026-08-04",
		"--page", "2",
		"--per-page", "25",
	})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, []map[string]any{{
		"type":        "NekiAdminChangeRequest",
		"id":          "change-1",
		"state":       "pending",
		"target_id":   "admin-1",
		"target_type": "NekiAdmin",
		"target_name": "Admin",
		"created_at":  "2026-08-04T12:00:00Z",
		"updated_at":  "2026-08-04T12:00:00Z",
		"admin_size":  "NKA_1",
	}})
}

func TestChangesListCmdRequiresTargetTypeForTargetID(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiChangesService{}
	cmd := changesListCmd(changesTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "--target-id", "admin-1"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, "--target-id requires --target-type")
	c.Assert(svc.ListFnInvoked, qt.IsFalse)
}

func TestChangesListCmdRejectsUnknownTargetType(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiChangesService{}
	cmd := changesListCmd(changesTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "--target-type", "tablet"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `unknown target type "tablet"; must be one of: admin, cluster, config-profile, router, sidecar`)
	c.Assert(svc.ListFnInvoked, qt.IsFalse)
}

func TestChangesListHumanDisplaysTarget(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	svc := &mock.NekiChangesService{ListFn: func(_ context.Context, req *ps.ListNekiChangesRequest) ([]*ps.NekiChange, error) {
		c.Assert(req, qt.DeepEquals, &ps.ListNekiChangesRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			PerPage:      100,
		})
		change := testChange()
		change.CreatedAt = time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
		return []*ps.NekiChange{change}, nil
	}}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiChanges: svc}, nil
		},
	}

	cmd := changesListCmd(ch)
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "TARGET")
	c.Assert(out.String(), qt.Contains, "admin")
	c.Assert(out.String(), qt.Contains, "Admin")
	c.Assert(out.String(), qt.Contains, "change-1")
}

func TestChangesShowCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiChangesService{GetFn: func(_ context.Context, req *ps.GetNekiChangeRequest) (*ps.NekiChange, error) {
		c.Assert(req, qt.DeepEquals, &ps.GetNekiChangeRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			Change:       "change-1",
		})
		return testChange(), nil
	}}

	cmd := changesShowCmd(changesTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "change-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, map[string]any{
		"type":        "NekiAdminChangeRequest",
		"id":          "change-1",
		"state":       "pending",
		"target_id":   "admin-1",
		"target_type": "NekiAdmin",
		"target_name": "Admin",
		"created_at":  "2026-08-04T12:00:00Z",
		"updated_at":  "2026-08-04T12:00:00Z",
		"admin_size":  "NKA_1",
	})
}

func TestChangesCancelCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiChangesService{CancelFn: func(_ context.Context, req *ps.CancelNekiChangeRequest) error {
		c.Assert(req, qt.DeepEquals, &ps.CancelNekiChangeRequest{
			Organization: "acme",
			Database:     "app",
			Branch:       "main",
			Change:       "change-1",
		})
		return nil
	}}

	cmd := changesCancelCmd(changesTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "change-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, map[string]any{
		"result": "change canceled", "change_id": "change-1",
	})
}

func TestChangesShowNotFound(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiChangesService{GetFn: func(context.Context, *ps.GetNekiChangeRequest) (*ps.NekiChange, error) {
		return nil, &ps.Error{Code: ps.ErrNotFound}
	}}

	cmd := changesShowCmd(changesTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "missing"})
	c.Assert(cmd.Execute(), qt.ErrorMatches, `(?s).*change.*missing.*does not exist.*`)
}

func TestNormalizeTargetType(t *testing.T) {
	c := qt.New(t)
	got, err := normalizeTargetTypes([]string{"NekiRouter", "config-profile", "SIDECAR"})
	c.Assert(err, qt.IsNil)
	c.Assert(got, qt.DeepEquals, []string{"NekiRouter", "NekiConfigurationProfile", "NekiSidecar"})
}
