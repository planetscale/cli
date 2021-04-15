package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type BackupsService struct {
	CreateFn        func(context.Context, *ps.CreateBackupRequest) (*ps.Backup, error)
	CreateFnInvoked bool

	GetFn        func(context.Context, *ps.GetBackupRequest) (*ps.Backup, error)
	GetFnInvoked bool

	ListFn        func(context.Context, *ps.ListBackupsRequest) ([]*ps.Backup, error)
	ListFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeleteBackupRequest) error
	DeleteFnInvoked bool
}

func (b *BackupsService) Create(ctx context.Context, req *ps.CreateBackupRequest) (*ps.Backup, error) {
	b.CreateFnInvoked = true
	return b.CreateFn(ctx, req)
}

func (b *BackupsService) List(ctx context.Context, req *ps.ListBackupsRequest) ([]*ps.Backup, error) {
	b.ListFnInvoked = true
	return b.ListFn(ctx, req)
}

func (b *BackupsService) Get(ctx context.Context, req *ps.GetBackupRequest) (*ps.Backup, error) {
	b.GetFnInvoked = true
	return b.GetFn(ctx, req)
}

func (b *BackupsService) Delete(ctx context.Context, req *ps.DeleteBackupRequest) error {
	b.DeleteFnInvoked = true
	return b.DeleteFn(ctx, req)
}
