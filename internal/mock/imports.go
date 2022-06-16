package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type DataImportsService struct {
	TestDataImportSourceFn          func(ctx context.Context, request *ps.TestDataImportSourceRequest) (*ps.TestDataImportSourceResponse, error)
	TestDataImportSourceFnInvoked   bool
	StartDataImportFn               func(ctx context.Context, request *ps.StartDataImportRequest) (*ps.DataImport, error)
	StartDataImportFnInvoked        bool
	CancelDataImportFn              func(ctx context.Context, request *ps.CancelDataImportRequest) error
	CancelDataImportFnInvoked       bool
	GetDataImportStatusFn           func(ctx context.Context, request *ps.GetImportStatusRequest) (*ps.DataImport, error)
	GetDataImportStatusFnInvoked    bool
	MakePlanetScalePrimaryFn        func(ctx context.Context, request *ps.MakePlanetScalePrimaryRequest) (*ps.DataImport, error)
	MakePlanetScalePrimaryFnInvoked bool
	MakePlanetScaleReplicaFn        func(ctx context.Context, request *ps.MakePlanetScaleReplicaRequest) (*ps.DataImport, error)
	MakePlanetScaleReplicaFnInvoked bool
	DetachExternalDatabaseFn        func(ctx context.Context, request *ps.DetachExternalDatabaseRequest) (*ps.DataImport, error)
	DetachExternalDatabaseFnInvoked bool
}

func (d *DataImportsService) TestDataImportSource(ctx context.Context, request *ps.TestDataImportSourceRequest) (*ps.TestDataImportSourceResponse, error) {
	d.TestDataImportSourceFnInvoked = true
	return d.TestDataImportSourceFn(ctx, request)
}

func (d *DataImportsService) StartDataImport(ctx context.Context, request *ps.StartDataImportRequest) (*ps.DataImport, error) {
	d.StartDataImportFnInvoked = true
	return d.StartDataImportFn(ctx, request)
}

func (d *DataImportsService) CancelDataImport(ctx context.Context, request *ps.CancelDataImportRequest) error {
	d.CancelDataImportFnInvoked = true
	return d.CancelDataImportFn(ctx, request)
}

func (d *DataImportsService) GetDataImportStatus(ctx context.Context, request *ps.GetImportStatusRequest) (*ps.DataImport, error) {
	d.GetDataImportStatusFnInvoked = true
	return d.GetDataImportStatusFn(ctx, request)
}

func (d *DataImportsService) MakePlanetScalePrimary(ctx context.Context, request *ps.MakePlanetScalePrimaryRequest) (*ps.DataImport, error) {
	d.MakePlanetScalePrimaryFnInvoked = true
	return d.MakePlanetScalePrimaryFn(ctx, request)
}

func (d *DataImportsService) MakePlanetScaleReplica(ctx context.Context, request *ps.MakePlanetScaleReplicaRequest) (*ps.DataImport, error) {
	d.MakePlanetScaleReplicaFnInvoked = true
	return d.MakePlanetScaleReplicaFn(ctx, request)
}

func (d *DataImportsService) DetachExternalDatabase(ctx context.Context, request *ps.DetachExternalDatabaseRequest) (*ps.DataImport, error) {
	d.DetachExternalDatabaseFnInvoked = true
	return d.DetachExternalDatabaseFn(ctx, request)
}
