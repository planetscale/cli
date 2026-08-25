package billing

import (
	"bytes"
	"context"
	"net/url"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func testInvoice() *ps.Invoice {
	return &ps.Invoice{
		ID:                 "inv_123",
		Total:              "12.34",
		BillingPeriodStart: "2026-07-01",
		BillingPeriodEnd:   "2026-07-31",
		Paid:               true,
		Overdue:            false,
	}
}

func testInvoiceLineItem() *ps.InvoiceLineItem {
	return &ps.InvoiceLineItem{
		ID:               "li_1",
		Subtotal:         12.34,
		Description:      "PS_10",
		MetricName:       "ps_10",
		CloudflareBilled: false,
		DatabaseID:       "db_1",
		DatabaseName:     "mydb",
		Resource:         ps.InvoiceLineItemResource{ID: "branch_1", Name: "main"},
	}
}

func TestInvoiceListCmd_WalksPages(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	pages := []string{}
	next2 := 2
	svc := &mock.InvoicesService{
		ListFn: func(ctx context.Context, req *ps.ListInvoicesRequest, opts ...ps.ListOption) (*ps.InvoicePage, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			page, perPage := listOpts(c, opts)
			pages = append(pages, page)
			c.Assert(perPage, qt.Equals, "100")
			if page == "1" {
				return &ps.InvoicePage{
					Data:     []*ps.Invoice{{ID: "inv_1", Total: "1.00"}},
					NextPage: &next2,
				}, nil
			}
			c.Assert(page, qt.Equals, "2")
			return &ps.InvoicePage{Data: []*ps.Invoice{testInvoice()}}, nil
		},
	}

	cmd := ListInvoicesCmd(invoiceTestHelper(&buf, printer.JSON, svc))
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(pages, qt.DeepEquals, []string{"1", "2"})
	c.Assert(buf.String(), qt.Contains, "inv_1")
	c.Assert(buf.String(), qt.Contains, "inv_123")
}

func TestInvoiceListCmd_SinglePage(t *testing.T) {
	c := qt.New(t)

	calls := 0
	svc := &mock.InvoicesService{
		ListFn: func(ctx context.Context, req *ps.ListInvoicesRequest, opts ...ps.ListOption) (*ps.InvoicePage, error) {
			calls++
			page, perPage := listOpts(c, opts)
			c.Assert(page, qt.Equals, "2")
			c.Assert(perPage, qt.Equals, "10")
			return &ps.InvoicePage{Data: []*ps.Invoice{testInvoice()}}, nil
		},
	}

	cmd := ListInvoicesCmd(invoiceTestHelper(&bytes.Buffer{}, printer.JSON, svc))
	cmd.SetArgs([]string{"--page", "2", "--per-page", "10"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(calls, qt.Equals, 1)
}

func TestInvoiceShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	svc := &mock.InvoicesService{
		GetFn: func(ctx context.Context, req *ps.GetInvoiceRequest) (*ps.Invoice, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			c.Assert(req.Invoice, qt.Equals, "inv_123")
			return testInvoice(), nil
		},
	}

	cmd := ShowInvoiceCmd(invoiceTestHelper(&buf, printer.JSON, svc))
	cmd.SetArgs([]string{"inv_123"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, "inv_123")
}

func TestInvoiceLineItemsCmd_WalksPages(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	pages := []string{}
	next2 := 2
	svc := &mock.InvoicesService{
		ListLineItemsFn: func(ctx context.Context, req *ps.ListInvoiceLineItemsRequest, opts ...ps.ListOption) (*ps.InvoiceLineItemPage, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			c.Assert(req.Invoice, qt.Equals, "inv_123")
			page, perPage := listOpts(c, opts)
			pages = append(pages, page)
			c.Assert(perPage, qt.Equals, "100")
			if page == "1" {
				return &ps.InvoiceLineItemPage{
					Data:     []*ps.InvoiceLineItem{{ID: "li_0", DatabaseName: "other"}},
					NextPage: &next2,
				}, nil
			}
			return &ps.InvoiceLineItemPage{Data: []*ps.InvoiceLineItem{testInvoiceLineItem()}}, nil
		},
	}

	cmd := InvoiceLineItemsCmd(invoiceTestHelper(&buf, printer.JSON, svc))
	cmd.SetArgs([]string{"inv_123"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(pages, qt.DeepEquals, []string{"1", "2"})
	c.Assert(buf.String(), qt.Contains, "li_0")
	c.Assert(buf.String(), qt.Contains, "li_1")
}

func TestInvoiceLineItemsCmd_SinglePage(t *testing.T) {
	c := qt.New(t)

	calls := 0
	svc := &mock.InvoicesService{
		ListLineItemsFn: func(ctx context.Context, req *ps.ListInvoiceLineItemsRequest, opts ...ps.ListOption) (*ps.InvoiceLineItemPage, error) {
			calls++
			page, perPage := listOpts(c, opts)
			c.Assert(page, qt.Equals, "3")
			c.Assert(perPage, qt.Equals, "25")
			return &ps.InvoiceLineItemPage{Data: []*ps.InvoiceLineItem{testInvoiceLineItem()}}, nil
		},
	}

	cmd := InvoiceLineItemsCmd(invoiceTestHelper(&bytes.Buffer{}, printer.JSON, svc))
	cmd.SetArgs([]string{"inv_123", "--page", "3", "--per-page", "25"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(calls, qt.Equals, 1)
}

func listOpts(c *qt.C, opts []ps.ListOption) (page, perPage string) {
	listOpts := &ps.ListOptions{URLValues: &url.Values{}}
	for _, opt := range opts {
		c.Assert(opt(listOpts), qt.IsNil)
	}
	return listOpts.URLValues.Get("page"), listOpts.URLValues.Get("per_page")
}

func invoiceTestHelper(buf *bytes.Buffer, format printer.Format, svc *mock.InvoicesService) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Invoices: svc}, nil
		},
	}
}
