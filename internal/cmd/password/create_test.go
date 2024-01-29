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

func Test_ttlFlag(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  time.Duration
		err  string
	}{
		{
			name: "empty",
			in:   "",
			out:  0,
		},
		{
			name: "zero",
			in:   "0",
			out:  0,
		},
		{
			name: "zero seconds",
			in:   "0s",
			out:  0,
		},
		{
			name: "invalid",
			in:   "x",
			err:  `cannot parse "x" as TTL in seconds`,
		},
		{
			name: "invalid duration",
			in:   "1x",
			err:  `cannot parse "1x" as TTL in seconds`,
		},
		{
			name: "negative",
			in:   "-1",
			err:  "TTL cannot be negative",
		},
		{
			name: "negative seconds",
			in:   "-1s",
			err:  "TTL cannot be negative",
		},
		{
			name: "negative",
			in:   "-1",
			err:  "TTL cannot be negative",
		},
		{
			name: "milliseconds",
			in:   "10ms",
			err:  "TTL must be defined in increments of 1 second",
		},
		{
			name: "rounding",
			in:   "10.4s",
			err:  "TTL must be defined in increments of 1 second",
		},
		{
			name: "integer",
			in:   "3600",
			out:  1 * time.Hour,
		},
		{
			name: "seconds",
			in:   "30s",
			out:  30 * time.Second,
		},
		{
			name: "minutes",
			in:   "15m",
			out:  15 * time.Minute,
		},
		{
			name: "hours",
			in:   "12h",
			out:  12 * time.Hour,
		},
		{
			name: "invalid days",
			in:   "0.1d",
			err:  `cannot parse "0.1d" as TTL in seconds`,
		},
		{
			name: "days",
			in:   "180d",
			out:  180 * 24 * time.Hour,
		},
		{
			name: "unsupported weeks",
			in:   "1w",
			err:  `cannot parse "1w" as TTL in seconds`,
		},
		{
			name: "unsupported years",
			in:   "1y",
			err:  `cannot parse "1y" as TTL in seconds`,
		},
		{
			name: "complex",
			in:   "48h10m30s",
			out:  48*time.Hour + 10*time.Minute + 30*time.Second,
		},
		{
			name: "complex days",
			in:   "1d10h",
			err:  `cannot parse "1d10h" as TTL in seconds`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ttl ttlFlag
			err := ttl.Set(tt.in)
			if tt.err != "" {
				require.EqualError(t, err, tt.err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.out, ttl.Value)
		})
	}
}
