package password

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func TestPassword_UpdateCmdRenames(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"
	passwordID := "pscale_pw_xxx"

	res := &ps.DatabaseBranchPassword{
		PublicID: passwordID,
		Name:     "reporting",
		Branch:   ps.DatabaseBranch{Name: branch},
	}

	svc := &mock.PasswordsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateDatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.PasswordId, qt.Equals, passwordID)
			c.Assert(req.Name, qt.Equals, "reporting")
			c.Assert(req.CIDRs, qt.IsNil)
			return res, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Passwords: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{db, branch, passwordID, "--new-name", "reporting"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestPassword_UpdateCmdSetsCIDRs(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.PasswordsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateDatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error) {
			c.Assert(req.Name, qt.Equals, "")
			c.Assert(req.CIDRs, qt.IsNotNil)
			c.Assert(*req.CIDRs, qt.DeepEquals, []string{"10.0.0.0/8", "192.168.1.1/32"})
			return &ps.DatabaseBranchPassword{PublicID: "pscale_pw_xxx"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Passwords: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"planetscale", "development", "pscale_pw_xxx", "--cidrs", "10.0.0.0/8,192.168.1.1/32"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestPassword_UpdateCmdClearsCIDRs(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.PasswordsService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateDatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error) {
			c.Assert(req.CIDRs, qt.IsNotNil)
			c.Assert(*req.CIDRs, qt.HasLen, 0)
			return &ps.DatabaseBranchPassword{PublicID: "pscale_pw_xxx"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Passwords: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"planetscale", "development", "pscale_pw_xxx", "--cidrs", ""})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestPassword_UpdateCmdByName(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	passwordID := "pscale_pw_xxx"

	svc := &mock.PasswordsService{
		ListFn: func(ctx context.Context, req *ps.ListDatabaseBranchPasswordRequest, opts ...ps.ListOption) ([]*ps.DatabaseBranchPassword, error) {
			return []*ps.DatabaseBranchPassword{{PublicID: passwordID, Name: "reporting"}}, nil
		},
		UpdateFn: func(ctx context.Context, req *ps.UpdateDatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error) {
			c.Assert(req.PasswordId, qt.Equals, passwordID)
			c.Assert(req.Name, qt.Equals, "analytics")
			return &ps.DatabaseBranchPassword{PublicID: passwordID, Name: "analytics"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Passwords: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"planetscale", "development", "--name", "reporting", "--new-name", "analytics"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestPassword_UpdateCmdRequiresAFlag(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.PasswordsService{}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Passwords: svc}, nil
		},
	}

	cmd := UpdateCmd(ch)
	cmd.SetArgs([]string{"planetscale", "development", "pscale_pw_xxx"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "at least one of --new-name or --cidrs")
	c.Assert(svc.UpdateFnInvoked, qt.IsFalse)
}
