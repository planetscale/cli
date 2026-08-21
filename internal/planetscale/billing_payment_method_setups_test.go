package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBillingPaymentMethodSetupsCreateAndGet(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/v1/organizations/my-org/billing/payment-method-setups" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"pmsetup1","state":"pending","checkout_url":"https://checkout.stripe.com/test"}`))
		case 2:
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/v1/organizations/my-org/billing/payment-method-setups/pmsetup1" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"id":"pmsetup1","state":"completed","checkout_url":null}`))
		default:
			t.Fatalf("unexpected request %d", requests)
		}
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	setup, err := client.PaymentMethodSetups.Create(context.Background(), &CreateBillingPaymentMethodSetupRequest{
		Organization: "my-org",
	})
	if err != nil {
		t.Fatal(err)
	}
	if setup.ID != "pmsetup1" || setup.State != "pending" {
		t.Fatalf("setup = %#v", setup)
	}

	setup, err = client.PaymentMethodSetups.Get(context.Background(), &GetBillingPaymentMethodSetupRequest{
		Organization: "my-org",
		Setup:        "pmsetup1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if setup.State != "completed" || setup.CheckoutURL != "" {
		t.Fatalf("setup = %#v", setup)
	}
}
