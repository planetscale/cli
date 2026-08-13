package deployrequest

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestDeployRequest_QueueCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	deployment := &ps.Deployment{ID: "dep-1", DeployRequestNumber: 7, State: "queued"}

	svc := &mock.DeployRequestsService{
		GetDeployQueueFn: func(ctx context.Context, req *ps.GetDeployQueueRequest) ([]*ps.Deployment, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return []*ps.Deployment{deployment}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := QueueCmd(ch)
	cmd.SetArgs([]string{db})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetDeployQueueFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, []*ps.Deployment{deployment})
}

func TestDeployRequest_OperationsCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	op := &ps.DeployOperation{ID: "op-1", State: "pending", Keyspace: "main", Table: "users", Operation: "ALTER"}

	svc := &mock.DeployRequestsService{
		GetDeployOperationsFn: func(ctx context.Context, req *ps.GetDeployOperationsRequest) ([]*ps.DeployOperation, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, uint64(42))
			return []*ps.DeployOperation{op}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := OperationsCmd(ch)
	cmd.SetArgs([]string{db, "42"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetDeployOperationsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, []*ps.DeployOperation{op})
}

func TestDeployRequest_ReviewsCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	review := &ps.DeployRequestReview{ID: "rev-1", State: "approved", Body: "lgtm", Actor: ps.Actor{Name: "gomez"}}

	svc := &mock.DeployRequestsService{
		ListReviewsFn: func(ctx context.Context, req *ps.ListDeployRequestReviewsRequest) ([]*ps.DeployRequestReview, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, uint64(42))
			return []*ps.DeployRequestReview{review}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := ReviewsCmd(ch)
	cmd.SetArgs([]string{db, "42"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListReviewsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, []*ps.DeployRequestReview{review})
}

func TestDeployRequest_DeploymentCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	deployment := &ps.Deployment{ID: "dep-1", DeployRequestNumber: 42, State: "in_progress", AutoCutover: true}

	svc := &mock.DeployRequestsService{
		GetDeploymentFn: func(ctx context.Context, req *ps.GetDeploymentRequest) (*ps.Deployment, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, uint64(42))
			return deployment, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := DeploymentCmd(ch)
	cmd.SetArgs([]string{db, "42"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetDeploymentFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, deployment)
}

func TestDeployRequest_StorageCheckCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	check := &ps.DeployRequestStorageCheck{EnoughStorage: true, Upgradeable: false, StorageBytesNeeded: 0}

	svc := &mock.DeployRequestsService{
		CheckStorageFn: func(ctx context.Context, req *ps.CheckDeployRequestStorageRequest) (*ps.DeployRequestStorageCheck, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, uint64(42))
			return check, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := StorageCheckCmd(ch)
	cmd.SetArgs([]string{db, "42"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CheckStorageFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, check)
}

func TestDeployRequest_ThrottlerShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	throttler := &ps.DeployRequestThrottler{
		Keyspaces: []string{"main"},
		Configurations: []*ps.ThrottlerConfiguration{
			{KeyspaceName: "main", Ratio: 50},
		},
	}

	svc := &mock.DeployRequestsService{
		GetThrottlerFn: func(ctx context.Context, req *ps.GetDeployRequestThrottlerRequest) (*ps.DeployRequestThrottler, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, uint64(42))
			return throttler, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := ThrottlerShowCmd(ch)
	cmd.SetArgs([]string{db, "42"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetThrottlerFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, throttler)
}

func TestDeployRequest_ThrottlerUpdateCmd_Ratio(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	throttler := &ps.DeployRequestThrottler{
		Keyspaces: []string{"main"},
		Configurations: []*ps.ThrottlerConfiguration{
			{KeyspaceName: "main", Ratio: 25},
		},
	}

	svc := &mock.DeployRequestsService{
		UpdateThrottlerFn: func(ctx context.Context, req *ps.UpdateDeployRequestThrottlerRequest) (*ps.DeployRequestThrottler, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, uint64(42))
			c.Assert(req.Ratio, qt.IsNotNil)
			c.Assert(*req.Ratio, qt.Equals, 25)
			c.Assert(req.Configurations, qt.IsNil)
			return throttler, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := ThrottlerUpdateCmd(ch)
	cmd.SetArgs([]string{db, "42", "--ratio", "25"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateThrottlerFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, throttler)
}

func TestDeployRequest_ThrottlerUpdateCmd_Configurations(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	throttler := &ps.DeployRequestThrottler{
		Keyspaces: []string{"main", "sharded"},
		Configurations: []*ps.ThrottlerConfiguration{
			{KeyspaceName: "main", Ratio: 10},
			{KeyspaceName: "sharded", Ratio: 40},
		},
	}

	svc := &mock.DeployRequestsService{
		UpdateThrottlerFn: func(ctx context.Context, req *ps.UpdateDeployRequestThrottlerRequest) (*ps.DeployRequestThrottler, error) {
			c.Assert(req.Ratio, qt.IsNil)
			c.Assert(req.Configurations, qt.DeepEquals, []*ps.UpdateThrottlerConfiguration{
				{KeyspaceName: "main", Ratio: 10},
				{KeyspaceName: "sharded", Ratio: 40},
			})
			return throttler, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := ThrottlerUpdateCmd(ch)
	cmd.SetArgs([]string{db, "42", "--configuration", "main=10", "--configuration", "sharded=40"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateThrottlerFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, throttler)
}

func TestDeployRequest_ThrottlerUpdateCmd_RequiresFlags(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: &mock.DeployRequestsService{}}, nil
		},
	}

	cmd := ThrottlerUpdateCmd(ch)
	cmd.SetArgs([]string{"db", "42"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, "must specify --ratio or --configuration")
}

func TestDeployRequest_ThrottlerUpdateCmd_RejectsCombinedModes(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: &mock.DeployRequestsService{}}, nil
		},
	}

	cmd := ThrottlerUpdateCmd(ch)
	cmd.SetArgs([]string{"db", "42", "--ratio", "25", "--configuration", "main=10"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, "cannot use both --ratio and --configuration; pick one mode")
}
