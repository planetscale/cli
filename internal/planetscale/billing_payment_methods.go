package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
)

type GetBillingPaymentMethodRequest struct {
	Organization string
}

type DeleteBillingPaymentMethodRequest struct {
	Organization string
}

// BillingPaymentMethod is the organization's current card.
type BillingPaymentMethod struct {
	ID       string `json:"id"`
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
	Name     string `json:"name,omitempty"`
}

type BillingPaymentMethodsService interface {
	Get(context.Context, *GetBillingPaymentMethodRequest) (*BillingPaymentMethod, error)
	Delete(context.Context, *DeleteBillingPaymentMethodRequest) error
}

type billingPaymentMethodsService struct {
	client *Client
}

var _ BillingPaymentMethodsService = &billingPaymentMethodsService{}

func (s *billingPaymentMethodsService) Get(ctx context.Context, getReq *GetBillingPaymentMethodRequest) (*BillingPaymentMethod, error) {
	req, err := s.client.newRequest(http.MethodGet, billingPaymentMethodAPIPath(getReq.Organization), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	card := &BillingPaymentMethod{}
	if err := s.client.do(ctx, req, card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *billingPaymentMethodsService) Delete(ctx context.Context, deleteReq *DeleteBillingPaymentMethodRequest) error {
	req, err := s.client.newRequest(http.MethodDelete, billingPaymentMethodAPIPath(deleteReq.Organization), nil)
	if err != nil {
		return fmt.Errorf("error creating http request: %w", err)
	}

	return s.client.do(ctx, req, nil)
}

func billingPaymentMethodAPIPath(org string) string {
	return path.Join("v1/organizations", org, "billing/payment-method")
}
