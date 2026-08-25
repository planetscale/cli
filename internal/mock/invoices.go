package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type InvoicesService struct {
	ListFn                 func(context.Context, *ps.ListInvoicesRequest, ...ps.ListOption) (*ps.InvoicePage, error)
	ListFnInvoked          bool
	GetFn                  func(context.Context, *ps.GetInvoiceRequest) (*ps.Invoice, error)
	GetFnInvoked           bool
	ListLineItemsFn        func(context.Context, *ps.ListInvoiceLineItemsRequest, ...ps.ListOption) (*ps.InvoiceLineItemPage, error)
	ListLineItemsFnInvoked bool
}

func (s *InvoicesService) List(ctx context.Context, req *ps.ListInvoicesRequest, opts ...ps.ListOption) (*ps.InvoicePage, error) {
	s.ListFnInvoked = true
	return s.ListFn(ctx, req, opts...)
}

func (s *InvoicesService) Get(ctx context.Context, req *ps.GetInvoiceRequest) (*ps.Invoice, error) {
	s.GetFnInvoked = true
	return s.GetFn(ctx, req)
}

func (s *InvoicesService) ListLineItems(ctx context.Context, req *ps.ListInvoiceLineItemsRequest, opts ...ps.ListOption) (*ps.InvoiceLineItemPage, error) {
	s.ListLineItemsFnInvoked = true
	return s.ListLineItemsFn(ctx, req, opts...)
}
