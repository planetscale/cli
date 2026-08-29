package user

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestUser_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	user := &ps.User{
		ID:                      "user-123",
		DisplayName:             "Ada Lovelace",
		Name:                    "Ada",
		Email:                   "ada@example.com",
		AvatarURL:               "https://example.com/avatar.png",
		TwoFactorAuthConfigured: true,
		CreatedAt:               createdAt,
	}

	svc := &mock.UsersService{
		GetCurrentUserFn: func(ctx context.Context) (*ps.User, error) {
			return user, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Users: svc}, nil
		},
	}

	cmd := ShowCmd(ch)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetCurrentUserFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &User{orig: user})
}

func TestUser_ShowCmd_Error(t *testing.T) {
	c := qt.New(t)

	wantErr := errors.New("request failed")
	format := printer.JSON
	svc := &mock.UsersService{
		GetCurrentUserFn: func(ctx context.Context) (*ps.User, error) {
			return nil, wantErr
		},
	}

	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Users: svc}, nil
		},
	}

	err := ShowCmd(ch).Execute()

	c.Assert(err, qt.ErrorIs, wantErr)
}
