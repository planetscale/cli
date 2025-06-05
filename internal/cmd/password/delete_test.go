package password

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	qt "github.com/frankban/quicktest"
)

func TestPassword_DeleteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"
	password := "mypassword"

	svc := &mock.PasswordsService{
		DeleteFn: func(ctx context.Context, req *ps.DeleteDatabaseBranchPasswordRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.PasswordId, qt.Equals, password)

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
				Passwords: svc,
			}, nil
		},
	}

	cmd := DeleteCmd(ch)
	cmd.SetArgs([]string{db, branch, password, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)

	res := map[string]string{
		"result":      "password deleted",
		"password_id": password,
		"branch":      branch,
	}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestPassword_DeleteCmdByName(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"
	passwordName := "test-password"
	passwordId := "password123"

	svc := &mock.PasswordsService{
		ListFn: func(ctx context.Context, req *ps.ListDatabaseBranchPasswordRequest, opts ...ps.ListOption) ([]*ps.DatabaseBranchPassword, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)

			return []*ps.DatabaseBranchPassword{
				{
					PublicID: passwordId,
					Name:     passwordName,
				},
				{
					PublicID: "other-id",
					Name:     "other-password",
				},
			}, nil
		},
		DeleteFn: func(ctx context.Context, req *ps.DeleteDatabaseBranchPasswordRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.PasswordId, qt.Equals, passwordId)

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
				Passwords: svc,
			}, nil
		},
	}

	cmd := DeleteCmd(ch)
	cmd.SetArgs([]string{db, branch, "--name", passwordName, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)

	res := map[string]string{
		"result":      "password deleted",
		"password_id": passwordId,
		"branch":      branch,
	}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestPassword_DeleteCmdByNameNotFound(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"
	passwordName := "nonexistent-password"

	svc := &mock.PasswordsService{
		ListFn: func(ctx context.Context, req *ps.ListDatabaseBranchPasswordRequest, opts ...ps.ListOption) ([]*ps.DatabaseBranchPassword, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)

			return []*ps.DatabaseBranchPassword{
				{
					PublicID: "other-id",
					Name:     "other-password",
				},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Passwords: svc,
			}, nil
		},
	}

	cmd := DeleteCmd(ch)
	cmd.SetArgs([]string{db, branch, "--name", passwordName, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `password with name nonexistent-password does not exist.*`)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
}