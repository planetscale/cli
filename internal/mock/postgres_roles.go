package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type PostgresRolesService struct {
	ListFn                    func(context.Context, *ps.ListPostgresRolesRequest, ...ps.ListOption) ([]*ps.PostgresRole, error)
	ListFnInvoked             bool
	GetFn                     func(context.Context, *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error)
	GetFnInvoked              bool
	CreateFn                  func(context.Context, *ps.CreatePostgresRoleRequest) (*ps.PostgresRole, error)
	CreateFnInvoked           bool
	UpdateFn                  func(context.Context, *ps.UpdatePostgresRoleRequest) (*ps.PostgresRole, error)
	UpdateFnInvoked           bool
	RenewFn                   func(context.Context, *ps.RenewPostgresRoleRequest) (*ps.PostgresRole, error)
	RenewFnInvoked            bool
	DeleteFn                  func(context.Context, *ps.DeletePostgresRoleRequest) error
	DeleteFnInvoked           bool
	ResetDefaultRoleFn        func(context.Context, *ps.ResetDefaultRoleRequest) (*ps.PostgresRole, error)
	ResetDefaultRoleFnInvoked bool
}

func (s *PostgresRolesService) List(ctx context.Context, req *ps.ListPostgresRolesRequest, opts ...ps.ListOption) ([]*ps.PostgresRole, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}

func (s *PostgresRolesService) Get(ctx context.Context, req *ps.GetPostgresRoleRequest) (*ps.PostgresRole, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *PostgresRolesService) Create(ctx context.Context, req *ps.CreatePostgresRoleRequest) (*ps.PostgresRole, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *PostgresRolesService) Update(ctx context.Context, req *ps.UpdatePostgresRoleRequest) (*ps.PostgresRole, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *PostgresRolesService) Renew(ctx context.Context, req *ps.RenewPostgresRoleRequest) (*ps.PostgresRole, error) {
	s.RenewFnInvoked = true
	return s.RenewFn(ctx, req)
}

func (s *PostgresRolesService) Delete(ctx context.Context, req *ps.DeletePostgresRoleRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}

func (s *PostgresRolesService) ResetDefaultRole(ctx context.Context, req *ps.ResetDefaultRoleRequest) (*ps.PostgresRole, error) {
	s.ResetDefaultRoleFnInvoked = true
	return s.ResetDefaultRoleFn(ctx, req)
}