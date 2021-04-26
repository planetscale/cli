package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type SchemaSnapshotsService struct {
	CreateFn        func(context.Context, *ps.CreateSchemaSnapshotRequest) (*ps.SchemaSnapshot, error)
	CreateFnInvoked bool

	ListFn        func(context.Context, *ps.ListSchemaSnapshotsRequest) ([]*ps.SchemaSnapshot, error)
	ListFnInvoked bool

	GetFn        func(context.Context, *ps.GetSchemaSnapshotRequest) (*ps.SchemaSnapshot, error)
	GetFnInvoked bool

	DiffFn        func(context.Context, *ps.DiffSchemaSnapshotRequest) ([]*ps.Diff, error)
	DiffFnInvoked bool
}

func (s *SchemaSnapshotsService) Create(ctx context.Context, req *ps.CreateSchemaSnapshotRequest) (*ps.SchemaSnapshot, error) {
	s.CreateFnInvoked = true
	return s.CreateFn(ctx, req)
}

func (s *SchemaSnapshotsService) List(ctx context.Context, req *ps.ListSchemaSnapshotsRequest) ([]*ps.SchemaSnapshot, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req)
}

func (s *SchemaSnapshotsService) Get(ctx context.Context, req *ps.GetSchemaSnapshotRequest) (*ps.SchemaSnapshot, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *SchemaSnapshotsService) Diff(ctx context.Context, req *ps.DiffSchemaSnapshotRequest) ([]*ps.Diff, error) {
	s.DiffFnInvoked = true
	return s.DiffFn(ctx, req)
}
