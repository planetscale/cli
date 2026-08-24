package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBillingPaymentMethodsGetAndDelete(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/v1/organizations/my-org/billing/payment-method" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"id":"pm_123","brand":"visa","last4":"4242","exp_month":12,"exp_year":2030}`))
		case 2:
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s, want DELETE", r.Method)
			}
			if r.URL.Path != "/v1/organizations/my-org/billing/payment-method" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %d", requests)
		}
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	card, err := client.PaymentMethods.Get(context.Background(), &GetBillingPaymentMethodRequest{
		Organization: "my-org",
	})
	if err != nil {
		t.Fatal(err)
	}
	if card.ID != "pm_123" || card.Brand != "visa" || card.Last4 != "4242" || card.ExpMonth != 12 || card.ExpYear != 2030 {
		t.Fatalf("card = %#v", card)
	}

	if err := client.PaymentMethods.Delete(context.Background(), &DeleteBillingPaymentMethodRequest{
		Organization: "my-org",
	}); err != nil {
		t.Fatal(err)
	}
}
