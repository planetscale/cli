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

func TestPassword_ShowCmd(t *testing.T) {
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
		Role:     "reader",
		CIDRs:    []string{"10.0.0.0/8"},
		Branch:   ps.DatabaseBranch{Name: branch},
	}

	svc := &mock.PasswordsService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.PasswordId, qt.Equals, passwordID)
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

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, branch, passwordID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestPassword_ShowCmdByName(t *testing.T) {
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
		ListFn: func(ctx context.Context, req *ps.ListDatabaseBranchPasswordRequest, opts ...ps.ListOption) ([]*ps.DatabaseBranchPassword, error) {
			return []*ps.DatabaseBranchPassword{res}, nil
		},
		GetFn: func(ctx context.Context, req *ps.GetDatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error) {
			c.Assert(req.PasswordId, qt.Equals, passwordID)
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

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, branch, "--name", "reporting"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
}

func TestPassword_ShowCmdRequiresSelector(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Passwords: &mock.PasswordsService{}}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{"planetscale", "development"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "must provide either password-id argument or --name flag")
}
