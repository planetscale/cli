package roleutil

import (
	"context"
	"fmt"
	"time"

	ps "github.com/planetscale/planetscale-go/planetscale"
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

// GenerateName generates a name for a temporary role with the given prefix
func GenerateName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().Unix())
}
