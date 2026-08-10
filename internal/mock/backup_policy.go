package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type BackupPoliciesService struct {
	ListFn        func(context.Context, *ps.ListBackupPoliciesRequest, ...ps.ListOption) ([]*ps.BackupPolicy, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetBackupPolicyRequest) (*ps.BackupPolicy, error)
	GetFnInvoked bool

	CreateFn        func(context.Context, *ps.CreateBackupPolicyRequest) (*ps.BackupPolicy, error)
	CreateFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdateBackupPolicyRequest) (*ps.BackupPolicy, error)
	UpdateFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeleteBackupPolicyRequest) error
	DeleteFnInvoked bool
}

func (s *BackupPoliciesService) List(ctx context.Context, req *ps.ListBackupPoliciesRequest, opts ...ps.ListOption) ([]*ps.BackupPolicy, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}

func (s *BackupPoliciesService) Get(ctx context.Context, req *ps.GetBackupPolicyRequest) (*ps.BackupPolicy, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *BackupPoliciesService) Create(ctx context.Context, req *ps.CreateBackupPolicyRequest) (*ps.BackupPolicy, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *BackupPoliciesService) Update(ctx context.Context, req *ps.UpdateBackupPolicyRequest) (*ps.BackupPolicy, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, req)
}

func (s *BackupPoliciesService) Delete(ctx context.Context, req *ps.DeleteBackupPolicyRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, req)
}
