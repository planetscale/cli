package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
)

type Invoice struct {
	ID                 string `json:"id"`
	Total              string `json:"total"`
	BillingPeriodStart string `json:"billing_period_start"`
	BillingPeriodEnd   string `json:"billing_period_end"`
	Paid               bool   `json:"paid"`
	Overdue            bool   `json:"overdue"`
}

type InvoiceLineItemResource struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	DeletedAt *string `json:"deleted_at"`
}

type InvoiceLineItem struct {
	ID               string                  `json:"id"`
	Subtotal         float64                 `json:"subtotal"`
	Description      string                  `json:"description"`
	MetricName       string                  `json:"metric_name"`
	CloudflareBilled bool                    `json:"cloudflare_billed"`
	DatabaseID       string                  `json:"database_id"`
	DatabaseName     string                  `json:"database_name"`
	Resource         InvoiceLineItemResource `json:"resource"`
}

type invoicesResponse struct {
	Data     []*Invoice `json:"data"`
	NextPage *int       `json:"next_page"`
}

type invoiceLineItemsResponse struct {
	Data     []*InvoiceLineItem `json:"data"`
	NextPage *int               `json:"next_page"`
}

type InvoicePage struct {
	Data     []*Invoice
	NextPage *int
}

type InvoiceLineItemPage struct {
	Data     []*InvoiceLineItem
	NextPage *int
}

type ListInvoicesRequest struct {
	Organization string
}

type GetInvoiceRequest struct {
	Organization string
	Invoice      string
}

type ListInvoiceLineItemsRequest struct {
	Organization string
	Invoice      string
}

type InvoicesService interface {
	List(context.Context, *ListInvoicesRequest, ...ListOption) (*InvoicePage, error)
	Get(context.Context, *GetInvoiceRequest) (*Invoice, error)
	ListLineItems(context.Context, *ListInvoiceLineItemsRequest, ...ListOption) (*InvoiceLineItemPage, error)
}

type invoicesService struct {
	client *Client
}

var _ InvoicesService = &invoicesService{}

func invoicesAPIPath(org string) string {
	return path.Join("v1/organizations", org, "invoices")
}

func invoiceAPIPath(org, invoice string) string {
	return path.Join(invoicesAPIPath(org), invoice)
}

func invoiceLineItemsAPIPath(org, invoice string) string {
	return path.Join(invoiceAPIPath(org, invoice), "line-items")
}

func (s *invoicesService) List(ctx context.Context, listReq *ListInvoicesRequest, opts ...ListOption) (*InvoicePage, error) {
	listOpts := defaultListOptions(opts...)
	req, err := s.client.newRequest(http.MethodGet, invoicesAPIPath(listReq.Organization), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list invoices: %w", err)
	}

	resp := &invoicesResponse{}
	if err := s.client.do(ctx, req, resp); err != nil {
		return nil, err
	}
	return &InvoicePage{Data: resp.Data, NextPage: resp.NextPage}, nil
}

func (s *invoicesService) Get(ctx context.Context, getReq *GetInvoiceRequest) (*Invoice, error) {
	req, err := s.client.newRequest(http.MethodGet, invoiceAPIPath(getReq.Organization, getReq.Invoice), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get invoice: %w", err)
	}

	invoice := &Invoice{}
	if err := s.client.do(ctx, req, invoice); err != nil {
		return nil, err
	}
	return invoice, nil
}

func (s *invoicesService) ListLineItems(ctx context.Context, listReq *ListInvoiceLineItemsRequest, opts ...ListOption) (*InvoiceLineItemPage, error) {
	listOpts := defaultListOptions(opts...)
	req, err := s.client.newRequest(http.MethodGet, invoiceLineItemsAPIPath(listReq.Organization, listReq.Invoice), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, fmt.Errorf("error creating request for list invoice line items: %w", err)
	}

	resp := &invoiceLineItemsResponse{}
	if err := s.client.do(ctx, req, resp); err != nil {
		return nil, err
	}
	return &InvoiceLineItemPage{Data: resp.Data, NextPage: resp.NextPage}, nil
}
