package password

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	qt "github.com/frankban/quicktest"
	"github.com/stretchr/testify/require"
)

func TestPassword_CreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"
	role := "reader"
	name := "production-password"
	res := &ps.DatabaseBranchPassword{Name: "foo"}

	svc := &mock.PasswordsService{
		CreateFn: func(ctx context.Context, req *ps.DatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, name)
			c.Assert(req.Role, qt.Equals, role)

			return res, nil
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

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch, name})
	cmd.Flag("role").Value.Set(role)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}

func TestPassword_CreateCmd_InvalidRole(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"
	name := "production-password"
	res := &ps.DatabaseBranchPassword{Name: "foo"}

	svc := &mock.PasswordsService{
		CreateFn: func(ctx context.Context, req *ps.DatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, name)
			c.Assert(req.Role, qt.Equals, "")

			return res, nil
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

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch, name})
	cmd.Flag("role").Value.Set("random")
	err := cmd.Execute()
	c.Assert(err, qt.IsNotNil)
	c.Assert(svc.CreateFnInvoked, qt.IsFalse)
}

func TestPassword_CreateCmd_DefaultRoleAdmin(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"
	name := "production-password"
	res := &ps.DatabaseBranchPassword{Name: "foo"}

	svc := &mock.PasswordsService{
		CreateFn: func(ctx context.Context, req *ps.DatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Name, qt.Equals, name)
			c.Assert(req.Role, qt.Equals, "admin")

			return res, nil
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

	cmd := CreateCmd(ch)
	cmd.SetArgs([]string{db, branch, name})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}

func Test_ttlSeconds(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		out  int
		err  string
	}{
		{
			name: "zero",
			in:   0 * time.Second,
			out:  0,
		},
		{
			name: "negative",
			in:   -1 * time.Second,
			err:  "TTL cannot be negative",
		},
		{
			name: "milliseconds",
			in:   10 * time.Millisecond,
			err:  "TTL must be at least 1 second",
		},
		{
			name: "rounding",
			in:   10*time.Second + 400*time.Millisecond,
			out:  10,
		},
		{
			name: "hours",
			in:   12 * time.Hour,
			out:  43200,
		},
		{
			name: "complex",
			in:   (7 * 24 * time.Hour) + 12*time.Hour + 10*time.Minute,
			out:  648600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := ttlSeconds(tt.in)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.out, out)
		})
	}
}
