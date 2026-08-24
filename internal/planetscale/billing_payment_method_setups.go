package planetscale

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"
)

type CreateBillingPaymentMethodSetupRequest struct {
	Organization string `json:"-"`
}

type GetBillingPaymentMethodSetupRequest struct {
	Organization string
	Setup        string
}

type BillingPaymentMethodSetup struct {
	ID          string     `json:"id"`
	State       string     `json:"state"`
	CheckoutURL string     `json:"checkout_url,omitempty"`
	Error       string     `json:"error,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
}

type BillingPaymentMethodSetupsService interface {
	Create(context.Context, *CreateBillingPaymentMethodSetupRequest) (*BillingPaymentMethodSetup, error)
	Get(context.Context, *GetBillingPaymentMethodSetupRequest) (*BillingPaymentMethodSetup, error)
}

type billingPaymentMethodSetupsService struct {
	client *Client
}

var _ BillingPaymentMethodSetupsService = &billingPaymentMethodSetupsService{}

func (s *billingPaymentMethodSetupsService) Create(ctx context.Context, createReq *CreateBillingPaymentMethodSetupRequest) (*BillingPaymentMethodSetup, error) {
	req, err := s.client.newRequest(http.MethodPost, billingPaymentMethodSetupsAPIPath(createReq.Organization), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	setup := &BillingPaymentMethodSetup{}
	if err := s.client.do(ctx, req, setup); err != nil {
		return nil, err
	}
	return setup, nil
}

func (s *billingPaymentMethodSetupsService) Get(ctx context.Context, getReq *GetBillingPaymentMethodSetupRequest) (*BillingPaymentMethodSetup, error) {
	req, err := s.client.newRequest(http.MethodGet, billingPaymentMethodSetupAPIPath(getReq.Organization, getReq.Setup), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	setup := &BillingPaymentMethodSetup{}
	if err := s.client.do(ctx, req, setup); err != nil {
		return nil, err
	}
	return setup, nil
}

func billingPaymentMethodSetupsAPIPath(org string) string {
	return path.Join("v1/organizations", org, "billing/payment-method-setups")
}

func billingPaymentMethodSetupAPIPath(org, setup string) string {
	return path.Join(billingPaymentMethodSetupsAPIPath(org), setup)
}
