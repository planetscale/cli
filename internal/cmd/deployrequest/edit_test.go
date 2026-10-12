package deployrequest

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestDeployRequest_EditCmdEnableAutoApply(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)
	enable := true

	svc := &mock.DeployRequestsService{
		AutoApplyFn: func(ctx context.Context, req *ps.AutoApplyDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Number, qt.Equals, number)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Enable, qt.Equals, enable)

			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := EditCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--enable-auto-apply"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.AutoApplyFnInvoked, qt.IsTrue)

	res := &ps.DeployRequest{Number: number}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDeployRequest_EditCmdDisableAutoApply(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)
	enable := false

	svc := &mock.DeployRequestsService{
		AutoApplyFn: func(ctx context.Context, req *ps.AutoApplyDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Number, qt.Equals, number)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Enable, qt.Equals, enable)

			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := EditCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--disable-auto-apply"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.AutoApplyFnInvoked, qt.IsTrue)

	res := &ps.DeployRequest{Number: number}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDeployRequest_EditCmdNoFlags(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{}, nil
		},
	}

	cmd := EditCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10)})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "must specify at least one of --enable-auto-apply, --disable-auto-apply, --auto-delete-branch, --aggressive-cutover, or --auto-apply")
}

func TestDeployRequest_EditCmdBothFlags(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{}, nil
		},
	}

	cmd := EditCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--enable-auto-apply", "--disable-auto-apply"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "cannot use both --enable-auto-apply and --disable-auto-apply flags together")
}

func TestDeployRequest_EditCmdDeprecatedAutoApplyEnable(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)
	enable := true

	svc := &mock.DeployRequestsService{
		AutoApplyFn: func(ctx context.Context, req *ps.AutoApplyDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Number, qt.Equals, number)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Enable, qt.Equals, enable)

			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := EditCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--auto-apply=enable"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.AutoApplyFnInvoked, qt.IsTrue)

	res := &ps.DeployRequest{Number: number}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDeployRequest_EditCmdDeprecatedAutoApplyDisable(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)
	enable := false

	svc := &mock.DeployRequestsService{
		AutoApplyFn: func(ctx context.Context, req *ps.AutoApplyDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Number, qt.Equals, number)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Enable, qt.Equals, enable)

			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := EditCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--auto-apply=disable"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.AutoApplyFnInvoked, qt.IsTrue)

	res := &ps.DeployRequest{Number: number}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDeployRequest_EditCmdMixedFlags(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)
	enable := true // Should use --enable-auto-apply and ignore --auto-apply

	svc := &mock.DeployRequestsService{
		AutoApplyFn: func(ctx context.Context, req *ps.AutoApplyDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Number, qt.Equals, number)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Enable, qt.Equals, enable)

			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := EditCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--enable-auto-apply", "--auto-apply=disable"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.AutoApplyFnInvoked, qt.IsTrue)

	res := &ps.DeployRequest{Number: number}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDeployRequest_UpdateCmdEnableAutoDeleteBranch(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)

	svc := &mock.DeployRequestsService{
		AutoDeleteBranchFn: func(ctx context.Context, req *ps.AutoDeleteBranchRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Number, qt.Equals, number)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Enable, qt.IsTrue)
			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--auto-delete-branch"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.AutoDeleteBranchFnInvoked, qt.IsTrue)
	c.Assert(svc.AutoApplyFnInvoked, qt.IsFalse)
	c.Assert(buf.String(), qt.JSONEquals, &ps.DeployRequest{Number: number})
}

func TestDeployRequest_UpdateCmdBothSettings(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)

	svc := &mock.DeployRequestsService{
		AutoApplyFn: func(ctx context.Context, req *ps.AutoApplyDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Enable, qt.IsTrue)
			return &ps.DeployRequest{Number: number}, nil
		},
		AutoDeleteBranchFn: func(ctx context.Context, req *ps.AutoDeleteBranchRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Enable, qt.IsFalse)
			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--enable-auto-apply", "--auto-delete-branch=false"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.AutoApplyFnInvoked, qt.IsTrue)
	c.Assert(svc.AutoDeleteBranchFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &ps.DeployRequest{Number: number})
}

func TestDeployRequest_UpdateCmdDisableAutoDeleteBranch(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.DeployRequestsService{
		AutoDeleteBranchFn: func(ctx context.Context, req *ps.AutoDeleteBranchRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Enable, qt.IsFalse)
			return &ps.DeployRequest{Number: 10}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"planetscale", "10", "--auto-delete-branch=false"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.AutoDeleteBranchFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &ps.DeployRequest{Number: 10})
}

func TestDeployRequest_UpdateCmdAggressiveCutoverFalse(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)

	svc := &mock.DeployRequestsService{
		AggressiveCutoverFn: func(ctx context.Context, req *ps.DeployRequestAggressiveCutoverRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Number, qt.Equals, number)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Enable, qt.IsFalse)
			return &ps.DeployRequest{
				Number: number,
				Deployment: &ps.Deployment{
					AggressiveCutover: false,
				},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--aggressive-cutover=false"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.AggressiveCutoverFnInvoked, qt.IsTrue)
	c.Assert(svc.AutoApplyFnInvoked, qt.IsFalse)
	c.Assert(svc.AutoDeleteBranchFnInvoked, qt.IsFalse)
	c.Assert(buf.String(), qt.JSONEquals, &ps.DeployRequest{
		Number: number,
		Deployment: &ps.Deployment{
			AggressiveCutover: false,
		},
	})
}

func TestDeployRequest_UpdateCmdAggressiveCutover(t *testing.T) {
	c := qt.New(t)

	var human bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&human)

	org := "planetscale"
	db := "planetscale"
	number := uint64(1284)

	svc := &mock.DeployRequestsService{
		AggressiveCutoverFn: func(ctx context.Context, req *ps.DeployRequestAggressiveCutoverRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Enable, qt.IsTrue)
			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DeployRequests: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--aggressive-cutover"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.AggressiveCutoverFnInvoked, qt.IsTrue)
	c.Assert(human.String(), qt.Contains, "Successfully updated aggressive-cutover for")
}
