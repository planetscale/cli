package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestInvoices_ListGetAndLineItems(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		switch r.URL.Path {
		case "/v1/organizations/my-org/invoices":
			c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
			c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "10")
			_, err := w.Write([]byte(`{
				"type": "list",
				"next_page": null,
				"data": [{
					"id": "inv_123",
					"total": "12.34",
					"billing_period_start": "2026-07-01",
					"billing_period_end": "2026-07-31",
					"paid": true,
					"overdue": false
				}]
			}`))
			c.Assert(err, qt.IsNil)
		case "/v1/organizations/my-org/invoices/inv_123":
			_, err := w.Write([]byte(`{
				"id": "inv_123",
				"total": "12.34",
				"billing_period_start": "2026-07-01",
				"billing_period_end": "2026-07-31",
				"paid": true,
				"overdue": false
			}`))
			c.Assert(err, qt.IsNil)
		case "/v1/organizations/my-org/invoices/inv_123/line-items":
			c.Assert(r.URL.Query().Get("page"), qt.Equals, "3")
			c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "25")
			_, err := w.Write([]byte(`{
				"type": "list",
				"next_page": 4,
				"data": [{
					"id": "li_1",
					"subtotal": 12.34,
					"description": "PS_10",
					"metric_name": "ps_10",
					"cloudflare_billed": false,
					"database_id": "db_1",
					"database_name": "mydb",
					"resource": {
						"id": "branch_1",
						"name": "main",
						"created_at": "2026-07-01T00:00:00.000Z",
						"updated_at": "2026-07-01T00:00:00.000Z",
						"deleted_at": null
					}
				}]
			}`))
			c.Assert(err, qt.IsNil)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	page, err := client.Invoices.List(context.Background(), &ListInvoicesRequest{
		Organization: "my-org",
	}, WithPage(2), WithPerPage(10))
	c.Assert(err, qt.IsNil)
	c.Assert(page.Data, qt.HasLen, 1)
	c.Assert(page.Data[0].ID, qt.Equals, "inv_123")
	c.Assert(page.Data[0].Total, qt.Equals, "12.34")
	c.Assert(page.Data[0].Paid, qt.IsTrue)
	c.Assert(page.NextPage, qt.IsNil)

	invoice, err := client.Invoices.Get(context.Background(), &GetInvoiceRequest{
		Organization: "my-org",
		Invoice:      "inv_123",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(invoice.BillingPeriodStart, qt.Equals, "2026-07-01")

	items, err := client.Invoices.ListLineItems(context.Background(), &ListInvoiceLineItemsRequest{
		Organization: "my-org",
		Invoice:      "inv_123",
	}, WithPage(3), WithPerPage(25))
	c.Assert(err, qt.IsNil)
	c.Assert(items.Data, qt.HasLen, 1)
	c.Assert(items.Data[0].ID, qt.Equals, "li_1")
	c.Assert(items.Data[0].Subtotal, qt.Equals, 12.34)
	c.Assert(items.Data[0].DatabaseName, qt.Equals, "mydb")
	c.Assert(items.Data[0].Resource.Name, qt.Equals, "main")
	c.Assert(*items.NextPage, qt.Equals, 4)
}
