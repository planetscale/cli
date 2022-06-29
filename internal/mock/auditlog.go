package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type AuditLogService struct {
	ListFn        func(context.Context, *ps.ListAuditLogsRequest, ...ps.ListOption) (*ps.CursorPaginatedResponse[*ps.AuditLog], error)
	ListFnInvoked bool
}

func (a *AuditLogService) List(ctx context.Context, req *ps.ListAuditLogsRequest, opts ...ps.ListOption) (*ps.CursorPaginatedResponse[*ps.AuditLog], error) {
	a.ListFnInvoked = true
	return a.ListFn(ctx, req, opts...)
}
