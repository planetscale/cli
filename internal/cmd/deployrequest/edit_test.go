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

	c.Assert(err, qt.ErrorMatches, "must specify at least one of --enable-auto-apply, --disable-auto-apply, --enable-auto-delete-branch, --disable-auto-delete-branch, or --auto-apply")
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
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--enable-auto-delete-branch"})
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
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10), "--enable-auto-apply", "--disable-auto-delete-branch"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.AutoApplyFnInvoked, qt.IsTrue)
	c.Assert(svc.AutoDeleteBranchFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &ps.DeployRequest{Number: number})
}

func TestDeployRequest_UpdateCmdBothAutoDeleteFlags(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"planetscale", "10", "--enable-auto-delete-branch", "--disable-auto-delete-branch"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, "cannot use both --enable-auto-delete-branch and --disable-auto-delete-branch flags together")
}
