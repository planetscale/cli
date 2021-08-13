package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type PasswordsService struct {
	CreateFn        func(context.Context, *ps.DatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error)
	CreateFnInvoked bool
	ListFn          func(context.Context, *ps.ListDatabaseBranchPasswordRequest) ([]*ps.DatabaseBranchPassword, error)
	ListFnInvoked   bool
	GetFn           func(context.Context, *ps.GetDatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error)
	GetFnInvoked    bool
	DeleteFn        func(context.Context, *ps.DeleteDatabaseBranchPasswordRequest) error
	DeleteFnInvoked bool
}

func (b *PasswordsService) Create(ctx context.Context, req *ps.DatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error) {
	b.CreateFnInvoked = true
	return b.CreateFn(ctx, req)
}

func (b *PasswordsService) List(ctx context.Context, req *ps.ListDatabaseBranchPasswordRequest) ([]*ps.DatabaseBranchPassword, error) {
	b.ListFnInvoked = true
	return b.ListFn(ctx, req)
}

func (b *PasswordsService) Get(ctx context.Context, req *ps.GetDatabaseBranchPasswordRequest) (*ps.DatabaseBranchPassword, error) {
	b.GetFnInvoked = true
	return b.GetFn(ctx, req)
}

func (b *PasswordsService) Delete(ctx context.Context, req *ps.DeleteDatabaseBranchPasswordRequest) error {
	b.DeleteFnInvoked = true
	return b.DeleteFn(ctx, req)
}
