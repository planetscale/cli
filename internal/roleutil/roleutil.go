package roleutil

import (
	"context"
	"fmt"
	"time"

	ps "github.com/planetscale/cli/internal/planetscale"
)

const (
	roleReadyPollInterval = time.Second
	roleReadyTimeout      = time.Minute
)

// Options represents the options to create a new Postgres role
type Options struct {
	Organization   string
	Database       string
	Branch         string
	Name           string
	TTL            time.Duration
	InheritedRoles []string
}

// Role represents a Postgres role with cleanup capabilities
type Role struct {
	Role   *ps.PostgresRole
	client *ps.Client
	opts   Options
}

// New creates a new temporary Postgres role
func New(ctx context.Context, client *ps.Client, opts Options) (*Role, error) {
	role, err := client.PostgresRoles.Create(ctx, &ps.CreatePostgresRoleRequest{
		Organization:   opts.Organization,
		Database:       opts.Database,
		Branch:         opts.Branch,
		Name:           opts.Name,
		TTL:            int(opts.TTL.Seconds()),
		InheritedRoles: opts.InheritedRoles,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return &Role{
		Role:   role,
		client: client,
		opts:   opts,
	}, nil
}

// WaitUntilReady waits until a temporary role can accept connections.
// The create response contains the only copy of the password, so this method
// keeps the original role and only updates its readiness state.
func (r *Role) WaitUntilReady(ctx context.Context) error {
	if r.Role.Ready {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, roleReadyTimeout)
	defer cancel()

	ticker := time.NewTicker(roleReadyPollInterval)
	defer ticker.Stop()

	return r.waitUntilReady(ctx, ticker.C)
}

func (r *Role) waitUntilReady(ctx context.Context, ticks <-chan time.Time) error {
	if r.Role.Ready {
		return nil
	}

	req := &ps.GetPostgresRoleRequest{
		Organization: r.opts.Organization,
		Database:     r.opts.Database,
		Branch:       r.opts.Branch,
		RoleId:       r.Role.ID,
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for role %s to become ready: %w", r.Role.ID, ctx.Err())
		case <-ticks:
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("waiting for role %s to become ready: %w", r.Role.ID, err)
			}

			role, err := r.client.PostgresRoles.Get(ctx, req)
			if err != nil {
				return fmt.Errorf("checking role readiness: %w", err)
			}
			if role.Ready {
				r.Role.Ready = true
				return nil
			}
		}
	}
}

// Cleanup deletes the temporary role with an optional successor
func (r *Role) Cleanup(ctx context.Context, successor string) error {
	return r.client.PostgresRoles.Delete(ctx, &ps.DeletePostgresRoleRequest{
		Organization: r.opts.Organization,
		Database:     r.opts.Database,
		Branch:       r.opts.Branch,
		RoleId:       r.Role.ID,
		Successor:    successor, // Empty string for no successor
	})
}
