package vtctld

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func newThrottlerTestHelper(org string, svc *mock.VtctldService) (*cmdutil.Helper, *bytes.Buffer) {
	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Vtctld: svc}, nil
		},
	}
	return ch, &buf
}

func TestThrottlerStatus(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	svc := &mock.VtctldService{
		GetThrottlerStatusFn: func(ctx context.Context, req *ps.VtctldGetThrottlerStatusRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.TabletAlias, qt.Equals, "zone1-0000000100")
			return json.RawMessage(`{"keyspace":"commerce","enabled":true}`), nil
		},
	}

	ch, _ := newThrottlerTestHelper(org, svc)

	cmd := ThrottlerStatusCmd(ch)
	cmd.SetArgs([]string{db, branch, "--tablet-alias", "zone1-0000000100"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetThrottlerStatusFnInvoked, qt.IsTrue)
}

func TestThrottlerStatus_RequiresTabletAlias(t *testing.T) {
	c := qt.New(t)

	svc := &mock.VtctldService{}
	ch, _ := newThrottlerTestHelper("my-org", svc)

	cmd := ThrottlerStatusCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(svc.GetThrottlerStatusFnInvoked, qt.IsFalse)
}

func TestThrottlerCheck(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	svc := &mock.VtctldService{
		CheckThrottlerFn: func(ctx context.Context, req *ps.VtctldCheckThrottlerRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.TabletAlias, qt.Equals, "zone1-0000000100")
			c.Assert(req.AppName, qt.Equals, "online-ddl")
			c.Assert(req.Scope, qt.Equals, "self")
			c.Assert(req.SkipRequestHeartbeats, qt.IsNotNil)
			c.Assert(*req.SkipRequestHeartbeats, qt.IsTrue)
			return json.RawMessage(`{"response_code":"THROTTLER_RESPONSE_CODE_OK"}`), nil
		},
	}

	ch, _ := newThrottlerTestHelper(org, svc)

	cmd := ThrottlerCheckCmd(ch)
	cmd.SetArgs([]string{db, branch,
		"--tablet-alias", "zone1-0000000100",
		"--app-name", "online-ddl",
		"--scope", "self",
		"--skip-request-heartbeats",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CheckThrottlerFnInvoked, qt.IsTrue)
}

func TestThrottlerCheck_OmitsUnsetOptionalBools(t *testing.T) {
	c := qt.New(t)

	svc := &mock.VtctldService{
		CheckThrottlerFn: func(ctx context.Context, req *ps.VtctldCheckThrottlerRequest) (json.RawMessage, error) {
			// Unset bool flags stay nil so the server applies its defaults.
			c.Assert(req.SkipRequestHeartbeats, qt.IsNil)
			c.Assert(req.OkIfNotExists, qt.IsNil)
			return json.RawMessage(`{"response_code":"THROTTLER_RESPONSE_CODE_OK"}`), nil
		},
	}

	ch, _ := newThrottlerTestHelper("my-org", svc)

	cmd := ThrottlerCheckCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch", "--tablet-alias", "zone1-0000000100"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CheckThrottlerFnInvoked, qt.IsTrue)
}

func TestThrottlerUpdateConfig(t *testing.T) {
	c := qt.New(t)

	org, db, branch := "my-org", "my-db", "my-branch"

	svc := &mock.VtctldService{
		UpdateThrottlerConfigFn: func(ctx context.Context, req *ps.VtctldUpdateThrottlerConfigRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Keyspace, qt.Equals, "commerce")
			c.Assert(req.Enabled, qt.IsNotNil)
			c.Assert(*req.Enabled, qt.IsTrue)
			c.Assert(req.Threshold, qt.IsNotNil)
			c.Assert(*req.Threshold, qt.Equals, 2.5)
			return json.RawMessage(`{}`), nil
		},
	}

	ch, _ := newThrottlerTestHelper(org, svc)

	cmd := ThrottlerUpdateConfigCmd(ch)
	cmd.SetArgs([]string{db, branch,
		"--keyspace", "commerce",
		"--enabled",
		"--threshold", "2.5",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateThrottlerConfigFnInvoked, qt.IsTrue)
}

func TestThrottlerUpdateConfig_DisableOmitsThreshold(t *testing.T) {
	c := qt.New(t)

	svc := &mock.VtctldService{
		UpdateThrottlerConfigFn: func(ctx context.Context, req *ps.VtctldUpdateThrottlerConfigRequest) (json.RawMessage, error) {
			c.Assert(req.Enabled, qt.IsNotNil)
			c.Assert(*req.Enabled, qt.IsFalse)
			c.Assert(req.Threshold, qt.IsNil)
			return json.RawMessage(`{}`), nil
		},
	}

	ch, _ := newThrottlerTestHelper("my-org", svc)

	cmd := ThrottlerUpdateConfigCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch", "--keyspace", "commerce", "--enabled=false"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateThrottlerConfigFnInvoked, qt.IsTrue)
}

func TestThrottlerUpdateConfig_RequiresMutation(t *testing.T) {
	c := qt.New(t)

	svc := &mock.VtctldService{}
	ch, _ := newThrottlerTestHelper("my-org", svc)

	cmd := ThrottlerUpdateConfigCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch", "--keyspace", "commerce"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(svc.UpdateThrottlerConfigFnInvoked, qt.IsFalse)
}

func TestThrottlerUpdateConfig_ThrottleAppWithMetrics(t *testing.T) {
	c := qt.New(t)

	svc := &mock.VtctldService{
		UpdateThrottlerConfigFn: func(ctx context.Context, req *ps.VtctldUpdateThrottlerConfigRequest) (json.RawMessage, error) {
			c.Assert(req.Enabled, qt.IsNil)
			c.Assert(len(req.Apps), qt.Equals, 1)
			c.Assert(req.Apps[0].Name, qt.Equals, "rowstreamer")
			c.Assert(req.Apps[0].Ratio, qt.IsNotNil)
			c.Assert(*req.Apps[0].Ratio, qt.Equals, 0.5)
			c.Assert(req.Apps[0].ExpireAt, qt.Not(qt.Equals), "")
			c.Assert(req.AppCheckedMetrics, qt.DeepEquals, map[string][]string{
				"rowstreamer": {"lag", "loadavg"},
			})
			return json.RawMessage(`{}`), nil
		},
	}

	ch, _ := newThrottlerTestHelper("my-org", svc)

	cmd := ThrottlerUpdateConfigCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch",
		"--keyspace", "commerce",
		"--throttle-app", "rowstreamer",
		"--throttle-app-ratio", "0.5",
		"--throttle-app-duration", "96h",
		"--app-name", "rowstreamer",
		"--app-metrics", "lag,loadavg",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateThrottlerConfigFnInvoked, qt.IsTrue)
}

func TestThrottlerUpdateConfig_UnthrottleApp(t *testing.T) {
	c := qt.New(t)

	svc := &mock.VtctldService{
		UpdateThrottlerConfigFn: func(ctx context.Context, req *ps.VtctldUpdateThrottlerConfigRequest) (json.RawMessage, error) {
			c.Assert(req.Enabled, qt.IsNil)
			c.Assert(req.UnthrottleApps, qt.DeepEquals, []string{"rowstreamer"})
			return json.RawMessage(`{}`), nil
		},
	}

	ch, _ := newThrottlerTestHelper("my-org", svc)

	cmd := ThrottlerUpdateConfigCmd(ch)
	cmd.SetArgs([]string{"my-db", "my-branch",
		"--keyspace", "commerce",
		"--unthrottle-app", "rowstreamer",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateThrottlerConfigFnInvoked, qt.IsTrue)
}
