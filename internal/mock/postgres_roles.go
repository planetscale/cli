package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type PostgresRolesService struct {
	ResetDefaultRoleFn        func(context.Context, *ps.ResetDefaultRoleRequest) (*ps.PostgresRole, error)
	ResetDefaultRoleFnInvoked bool
}

func (s *PostgresRolesService) ResetDefaultRole(ctx context.Context, req *ps.ResetDefaultRoleRequest) (*ps.PostgresRole, error) {
	s.ResetDefaultRoleFnInvoked = true
	return s.ResetDefaultRoleFn(ctx, req)
}