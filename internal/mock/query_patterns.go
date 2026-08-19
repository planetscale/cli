package mock

import (
	"context"
	"io"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type QueryPatternsService struct {
	CreateReportFn          func(context.Context, *ps.CreateQueryPatternsReportRequest) (*ps.QueryPatternsReport, error)
	CreateReportFnInvoked   bool
	ListReportsFn           func(context.Context, *ps.ListQueryPatternsReportsRequest, ...ps.ListOption) ([]*ps.QueryPatternsReport, error)
	ListReportsFnInvoked    bool
	GetReportFn             func(context.Context, *ps.GetQueryPatternsReportRequest) (*ps.QueryPatternsReport, error)
	GetReportFnInvoked      bool
	DeleteReportFn          func(context.Context, *ps.DeleteQueryPatternsReportRequest) error
	DeleteReportFnInvoked   bool
	DownloadReportFn        func(context.Context, *ps.DownloadQueryPatternsReportRequest) (io.ReadCloser, error)
	DownloadReportFnInvoked bool
}

func (s *QueryPatternsService) CreateReport(ctx context.Context, req *ps.CreateQueryPatternsReportRequest) (*ps.QueryPatternsReport, error) {
	s.CreateReportFnInvoked = true
	return s.CreateReportFn(ctx, req)
}

func (s *QueryPatternsService) ListReports(ctx context.Context, req *ps.ListQueryPatternsReportsRequest, opts ...ps.ListOption) ([]*ps.QueryPatternsReport, error) {
	s.ListReportsFnInvoked = true
	return s.ListReportsFn(ctx, req, opts...)
}

func (s *QueryPatternsService) GetReport(ctx context.Context, req *ps.GetQueryPatternsReportRequest) (*ps.QueryPatternsReport, error) {
	s.GetReportFnInvoked = true
	return s.GetReportFn(ctx, req)
}

func (s *QueryPatternsService) DeleteReport(ctx context.Context, req *ps.DeleteQueryPatternsReportRequest) error {
	s.DeleteReportFnInvoked = true
	return s.DeleteReportFn(ctx, req)
}

func (s *QueryPatternsService) DownloadReport(ctx context.Context, req *ps.DownloadQueryPatternsReportRequest) (io.ReadCloser, error) {
	s.DownloadReportFnInvoked = true
	return s.DownloadReportFn(ctx, req)
}
