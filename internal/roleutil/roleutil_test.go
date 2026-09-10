package roleutil

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestWaitUntilReadyReturnsImmediatelyWhenReady(t *testing.T) {
	role := newTestRole(t, &ps.PostgresRole{ID: "role-id", Ready: true}, func(context.Context, *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error) {
		t.Fatal("Get should not be called for a ready role")
		return nil, nil
	})

	if err := role.WaitUntilReady(t.Context()); err != nil {
		t.Fatalf("WaitUntilReady: %v", err)
	}
}

func TestWaitUntilReadyPollsUntilReady(t *testing.T) {
	getCalls := 0
	role := newTestRole(t, &ps.PostgresRole{ID: "role-id", Password: "secret"}, func(_ context.Context, req *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error) {
		getCalls++
		if req.Organization != "org" || req.Database != "db" || req.Branch != "main" || req.RoleId != "role-id" {
			t.Fatalf("unexpected request: %+v", req)
		}
		return &ps.PostgresRole{ID: "role-id", Ready: getCalls == 2}, nil
	})

	ticks := make(chan time.Time, 2)
	ticks <- time.Now()
	ticks <- time.Now()
	if err := role.waitUntilReady(t.Context(), ticks); err != nil {
		t.Fatalf("waitUntilReady: %v", err)
	}
	if getCalls != 2 {
		t.Fatalf("Get calls = %d, want 2", getCalls)
	}
	if !role.Role.Ready {
		t.Fatal("role was not marked ready")
	}
	if role.Role.Password != "secret" {
		t.Fatalf("password = %q, want create-response password", role.Role.Password)
	}
}

func TestWaitUntilReadyReturnsAPIError(t *testing.T) {
	wantErr := errors.New("api unavailable")
	role := newTestRole(t, &ps.PostgresRole{ID: "role-id"}, func(context.Context, *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error) {
		return nil, wantErr
	})

	ticks := make(chan time.Time, 1)
	ticks <- time.Now()
	err := role.waitUntilReady(t.Context(), ticks)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped API error", err)
	}
}

func TestWaitUntilReadyReturnsContextError(t *testing.T) {
	role := newTestRole(t, &ps.PostgresRole{ID: "role-id"}, func(context.Context, *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error) {
		t.Fatal("Get should not be called after cancellation")
		return nil, nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := role.waitUntilReady(ctx, make(chan time.Time))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
}

func TestWaitUntilReadyReturnsTimeoutError(t *testing.T) {
	role := newTestRole(t, &ps.PostgresRole{ID: "role-id"}, func(context.Context, *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error) {
		t.Fatal("Get should not be called after the deadline")
		return nil, nil
	})

	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()
	err := role.waitUntilReady(ctx, make(chan time.Time))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
}

func newTestRole(t *testing.T, created *ps.PostgresRole, getFn func(context.Context, *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error)) *Role {
	t.Helper()

	service := &mock.PostgresRolesService{GetFn: getFn}
	return &Role{
		Role:   created,
		client: &ps.Client{PostgresRoles: service},
		opts: Options{
			Organization: "org",
			Database:     "db",
			Branch:       "main",
		},
	}
}
