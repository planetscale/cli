package branch

import (
	"bytes"
	"context"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestBranch_VtgateResizeCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	branchName := "main"
	ts := time.Now()

	resize := &ps.BranchResizeRequest{
		ID:                  "wantid",
		State:               "pending",
		VTGateName:          "VTG_320",
		PreviousVTGateName:  "VTG_5",
		VTGateCount:         2,
		PreviousVTGateCount: 1,
		CreatedAt:           ts,
		UpdatedAt:           ts,
	}

	svc := &mock.DatabaseBranchesService{
		ResizeFn: func(ctx context.Context, req *ps.ResizeBranchRequest) (*ps.BranchResizeRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branchName)
			c.Assert(req.VTGateSize, qt.Equals, "VTG_320")
			c.Assert(*req.VTGateCount, qt.Equals, 2)
			return resize, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: svc,
			}, nil
		},
	}

	cmd := VtgateResizeCmd(ch)
	cmd.SetArgs([]string{db, branchName, "--vtgate-size", "VTG_320", "--vtgate-count", "2"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ResizeFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, resize)
}

func TestBranch_VtgateResizeCmd_SizeOnly(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	branchName := "main"
	ts := time.Now()

	resize := &ps.BranchResizeRequest{
		ID:                 "wantid",
		State:              "pending",
		VTGateName:         "VTG_1280",
		PreviousVTGateName: "VTG_20",
		CreatedAt:          ts,
		UpdatedAt:          ts,
	}

	svc := &mock.DatabaseBranchesService{
		ResizeFn: func(ctx context.Context, req *ps.ResizeBranchRequest) (*ps.BranchResizeRequest, error) {
			c.Assert(req.VTGateSize, qt.Equals, "VTG_1280")
			c.Assert(req.VTGateCount, qt.IsNil)
			c.Assert(req.VTGateMaxCount, qt.IsNil)
			c.Assert(req.VTGateAutoscaling, qt.IsNil)
			c.Assert(req.VTGateTargetCPUUtilization, qt.IsNil)
			return resize, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: svc,
			}, nil
		},
	}

	cmd := VtgateResizeCmd(ch)
	cmd.SetArgs([]string{db, branchName, "--vtgate-size", "VTG-1280"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ResizeFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, resize)
}

func TestBranch_VtgateResizeCmd_RequiresFlags(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: "planetscale",
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{}, nil
		},
	}

	cmd := VtgateResizeCmd(ch)
	cmd.SetArgs([]string{"mydb", "main"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "nothing to change")
}

func TestBranch_VtgateResizeCmd_Autoscaling(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	branchName := "main"
	ts := time.Now()
	maxCount := 8
	targetCPU := 50

	resize := &ps.BranchResizeRequest{
		ID:                         "wantid",
		State:                      "pending",
		VTGateName:                 "VTG_320",
		PreviousVTGateName:         "VTG_5",
		VTGateCount:                2,
		PreviousVTGateCount:        1,
		VTGateMaxCount:             &maxCount,
		VTGateAutoscaling:          true,
		VTGateTargetCPUUtilization: &targetCPU,
		CreatedAt:                  ts,
		UpdatedAt:                  ts,
	}

	svc := &mock.DatabaseBranchesService{
		ResizeFn: func(ctx context.Context, req *ps.ResizeBranchRequest) (*ps.BranchResizeRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branchName)
			c.Assert(req.VTGateSize, qt.Equals, "VTG_320")
			c.Assert(*req.VTGateCount, qt.Equals, 2)
			c.Assert(*req.VTGateMaxCount, qt.Equals, 8)
			c.Assert(*req.VTGateAutoscaling, qt.IsTrue)
			c.Assert(*req.VTGateTargetCPUUtilization, qt.Equals, 50)
			return resize, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: svc,
			}, nil
		},
	}

	cmd := VtgateResizeCmd(ch)
	cmd.SetArgs([]string{
		db, branchName,
		"--vtgate-size", "VTG_320",
		"--vtgate-count", "2",
		"--vtgate-max-count", "8",
		"--vtgate-autoscaling",
		"--vtgate-target-cpu-utilization", "50",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ResizeFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, resize)
}

func TestBranch_VtgateResizeCmd_DisableAutoscaling(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	branchName := "main"
	ts := time.Now()

	resize := &ps.BranchResizeRequest{
		ID:                "wantid",
		State:             "pending",
		VTGateAutoscaling: false,
		CreatedAt:         ts,
		UpdatedAt:         ts,
	}

	svc := &mock.DatabaseBranchesService{
		ResizeFn: func(ctx context.Context, req *ps.ResizeBranchRequest) (*ps.BranchResizeRequest, error) {
			c.Assert(req.VTGateSize, qt.Equals, "")
			c.Assert(req.VTGateCount, qt.IsNil)
			c.Assert(req.VTGateMaxCount, qt.IsNil)
			c.Assert(*req.VTGateAutoscaling, qt.IsFalse)
			c.Assert(req.VTGateTargetCPUUtilization, qt.IsNil)
			return resize, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: svc,
			}, nil
		},
	}

	cmd := VtgateResizeCmd(ch)
	cmd.SetArgs([]string{db, branchName, "--vtgate-autoscaling=false"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ResizeFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, resize)
}

func TestBranch_VtgateResizeStatusCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	branchName := "main"
	ts := time.Now()

	resize := &ps.BranchResizeRequest{
		ID:         "wantid",
		State:      "completed",
		VTGateName: "VTG_320",
		CreatedAt:  ts,
		UpdatedAt:  ts,
	}

	svc := &mock.DatabaseBranchesService{
		ResizeStatusFn: func(ctx context.Context, req *ps.BranchResizeStatusRequest) (*ps.BranchResizeRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branchName)
			return resize, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: svc,
			}, nil
		},
	}

	cmd := VtgateResizeStatusCmd(ch)
	cmd.SetArgs([]string{db, branchName})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ResizeStatusFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, resize)
}

func TestBranch_VtgateResizeCancelCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	branchName := "main"

	svc := &mock.DatabaseBranchesService{
		CancelResizeFn: func(ctx context.Context, req *ps.CancelBranchResizeRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branchName)
			return nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DatabaseBranches: svc,
			}, nil
		},
	}

	cmd := VtgateResizeCancelCmd(ch)
	cmd.SetArgs([]string{db, branchName})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.CancelResizeFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{
		"result": "canceled",
		"branch": branchName,
	})
}
