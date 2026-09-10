package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type NekiShardConfigurationProfilesService struct {
	ListFn               func(context.Context, *ps.ListNekiShardConfigurationProfilesRequest) ([]*ps.NekiShardConfigurationProfile, error)
	GetFn                func(context.Context, *ps.GetNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error)
	CreateFn             func(context.Context, *ps.CreateNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error)
	UpdateFn             func(context.Context, *ps.UpdateNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error)
	DeleteFn             func(context.Context, *ps.DeleteNekiShardConfigurationProfileRequest) error
	GetDefaultFn         func(context.Context, *ps.GetDefaultNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error)
	SetDefaultFn         func(context.Context, *ps.SetDefaultNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error)
	ListParametersFn     func(context.Context, *ps.ListNekiShardConfigurationProfileParametersRequest) ([]*ps.NekiParameter, error)
	ListExtensionsFn     func(context.Context, *ps.ListNekiShardConfigurationProfileExtensionsRequest) ([]*ps.NekiExtension, error)
	UpdateExtensionFn    func(context.Context, *ps.UpdateNekiShardConfigurationProfileExtensionRequest) (*ps.NekiExtension, error)
	RunMaintenanceFn     func(context.Context, *ps.RunNekiShardConfigurationProfileMaintenanceRequest) error
	RunBulkMaintenanceFn func(context.Context, *ps.RunNekiShardConfigurationProfilesMaintenanceRequest) error
	ListChangesFn        func(context.Context, *ps.ListNekiShardConfigurationProfileChangesRequest) ([]*ps.NekiShardConfigurationProfileChange, error)
	GetChangeFn          func(context.Context, *ps.GetNekiShardConfigurationProfileChangeRequest) (*ps.NekiShardConfigurationProfileChange, error)
	CancelChangeFn       func(context.Context, *ps.CancelNekiShardConfigurationProfileChangeRequest) error

	ListFnInvoked, GetFnInvoked, CreateFnInvoked, UpdateFnInvoked, DeleteFnInvoked bool
	GetDefaultFnInvoked, SetDefaultFnInvoked                                       bool
	ListParametersFnInvoked, ListExtensionsFnInvoked, UpdateExtensionFnInvoked     bool
	RunMaintenanceFnInvoked, RunBulkMaintenanceFnInvoked                           bool
	ListChangesFnInvoked, GetChangeFnInvoked, CancelChangeFnInvoked                bool
}

func (s *NekiShardConfigurationProfilesService) List(ctx context.Context, r *ps.ListNekiShardConfigurationProfilesRequest) ([]*ps.NekiShardConfigurationProfile, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) Get(ctx context.Context, r *ps.GetNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) Create(ctx context.Context, r *ps.CreateNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) Update(ctx context.Context, r *ps.UpdateNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
	s.UpdateFnInvoked = true
	return s.UpdateFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) Delete(ctx context.Context, r *ps.DeleteNekiShardConfigurationProfileRequest) error {
	s.DeleteFnInvoked = true
	return s.DeleteFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) GetDefault(ctx context.Context, r *ps.GetDefaultNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
	s.GetDefaultFnInvoked = true
	return s.GetDefaultFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) SetDefault(ctx context.Context, r *ps.SetDefaultNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
	s.SetDefaultFnInvoked = true
	return s.SetDefaultFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) ListParameters(ctx context.Context, r *ps.ListNekiShardConfigurationProfileParametersRequest) ([]*ps.NekiParameter, error) {
	s.ListParametersFnInvoked = true
	return s.ListParametersFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) ListExtensions(ctx context.Context, r *ps.ListNekiShardConfigurationProfileExtensionsRequest) ([]*ps.NekiExtension, error) {
	s.ListExtensionsFnInvoked = true
	return s.ListExtensionsFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) UpdateExtension(ctx context.Context, r *ps.UpdateNekiShardConfigurationProfileExtensionRequest) (*ps.NekiExtension, error) {
	s.UpdateExtensionFnInvoked = true
	return s.UpdateExtensionFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) RunMaintenance(ctx context.Context, r *ps.RunNekiShardConfigurationProfileMaintenanceRequest) error {
	s.RunMaintenanceFnInvoked = true
	return s.RunMaintenanceFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) RunBulkMaintenance(ctx context.Context, r *ps.RunNekiShardConfigurationProfilesMaintenanceRequest) error {
	s.RunBulkMaintenanceFnInvoked = true
	return s.RunBulkMaintenanceFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) ListChanges(ctx context.Context, r *ps.ListNekiShardConfigurationProfileChangesRequest) ([]*ps.NekiShardConfigurationProfileChange, error) {
	s.ListChangesFnInvoked = true
	return s.ListChangesFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) GetChange(ctx context.Context, r *ps.GetNekiShardConfigurationProfileChangeRequest) (*ps.NekiShardConfigurationProfileChange, error) {
	s.GetChangeFnInvoked = true
	return s.GetChangeFn(ctx, r)
}
func (s *NekiShardConfigurationProfilesService) CancelChange(ctx context.Context, r *ps.CancelNekiShardConfigurationProfileChangeRequest) error {
	s.CancelChangeFnInvoked = true
	return s.CancelChangeFn(ctx, r)
}
