package planetscale

import (
	"context"
	"net/http"
	"path"
	"time"
)

var _ WebhooksService = &webhooksService{}

// WebhooksService is an interface for communicating with the PlanetScale
// Webhooks API.
type WebhooksService interface {
	List(context.Context, *ListWebhooksRequest, ...ListOption) ([]*Webhook, error)
	Create(context.Context, *CreateWebhookRequest) (*Webhook, error)
	Get(context.Context, *GetWebhookRequest) (*Webhook, error)
	Update(context.Context, *UpdateWebhookRequest) (*Webhook, error)
	Delete(context.Context, *DeleteWebhookRequest) error
	Test(context.Context, *TestWebhookRequest) error
}

type webhooksResponse struct {
	Webhooks []*Webhook `json:"data"`
}

// Webhook represents a PlanetScale webhook.
type Webhook struct {
	ID                                  string    `json:"id"`
	URL                                 string    `json:"url"`
	Secret                              string    `json:"secret"`
	WebhookAuthorizationTokenConfigured bool      `json:"webhook_authorization_token_configured"`
	Enabled                             bool      `json:"enabled"`
	LastSentResult                      string    `json:"last_sent_result"`
	LastSentSuccess                     bool      `json:"last_sent_success"`
	LastSentAt                          time.Time `json:"last_sent_at"`
	CreatedAt                           time.Time `json:"created_at"`
	UpdatedAt                           time.Time `json:"updated_at"`
	Events                              []string  `json:"events"`
}

// ListWebhooksRequest is the request for listing webhooks.
type ListWebhooksRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
}

// CreateWebhookRequest is the request for creating a webhook.
type CreateWebhookRequest struct {
	Organization              string   `json:"-"`
	Database                  string   `json:"-"`
	URL                       string   `json:"url"`
	WebhookAuthorizationToken string   `json:"webhook_authorization_token,omitempty"`
	Enabled                   *bool    `json:"enabled,omitempty"`
	Events                    []string `json:"events,omitempty"`
}

// GetWebhookRequest is the request for getting a webhook.
type GetWebhookRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	ID           string `json:"-"`
}

// UpdateWebhookRequest is the request for updating a webhook.
type UpdateWebhookRequest struct {
	Organization                   string   `json:"-"`
	Database                       string   `json:"-"`
	ID                             string   `json:"-"`
	URL                            *string  `json:"url,omitempty"`
	WebhookAuthorizationToken      *string  `json:"webhook_authorization_token,omitempty"`
	ClearWebhookAuthorizationToken bool     `json:"clear_webhook_authorization_token,omitempty"`
	Enabled                        *bool    `json:"enabled,omitempty"`
	Events                         []string `json:"events,omitempty"`
}

// DeleteWebhookRequest is the request for deleting a webhook.
type DeleteWebhookRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	ID           string `json:"-"`
}

// TestWebhookRequest is the request for testing a webhook.
type TestWebhookRequest struct {
	Organization string `json:"-"`
	Database     string `json:"-"`
	ID           string `json:"-"`
}

type webhooksService struct {
	client *Client
}

func (w *webhooksService) List(ctx context.Context, listReq *ListWebhooksRequest, opts ...ListOption) ([]*Webhook, error) {
	listOpts := defaultListOptions(opts...)

	req, err := w.client.newRequest(http.MethodGet, webhooksAPIPath(listReq.Organization, listReq.Database), nil, WithQueryParams(*listOpts.URLValues))
	if err != nil {
		return nil, err
	}

	resp := &webhooksResponse{}
	if err := w.client.do(ctx, req, &resp); err != nil {
		return nil, err
	}

	return resp.Webhooks, nil
}

func (w *webhooksService) Create(ctx context.Context, createReq *CreateWebhookRequest) (*Webhook, error) {
	req, err := w.client.newRequest(http.MethodPost, webhooksAPIPath(createReq.Organization, createReq.Database), createReq)
	if err != nil {
		return nil, err
	}

	webhook := &Webhook{}
	if err := w.client.do(ctx, req, &webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

func (w *webhooksService) Get(ctx context.Context, getReq *GetWebhookRequest) (*Webhook, error) {
	req, err := w.client.newRequest(http.MethodGet, webhookAPIPath(getReq.Organization, getReq.Database, getReq.ID), nil)
	if err != nil {
		return nil, err
	}

	webhook := &Webhook{}
	if err := w.client.do(ctx, req, &webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

func (w *webhooksService) Update(ctx context.Context, updateReq *UpdateWebhookRequest) (*Webhook, error) {
	req, err := w.client.newRequest(http.MethodPatch, webhookAPIPath(updateReq.Organization, updateReq.Database, updateReq.ID), updateReq)
	if err != nil {
		return nil, err
	}

	webhook := &Webhook{}
	if err := w.client.do(ctx, req, &webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

func (w *webhooksService) Delete(ctx context.Context, deleteReq *DeleteWebhookRequest) error {
	req, err := w.client.newRequest(http.MethodDelete, webhookAPIPath(deleteReq.Organization, deleteReq.Database, deleteReq.ID), nil)
	if err != nil {
		return err
	}

	return w.client.do(ctx, req, nil)
}

func (w *webhooksService) Test(ctx context.Context, testReq *TestWebhookRequest) error {
	req, err := w.client.newRequest(http.MethodPost, webhookTestAPIPath(testReq.Organization, testReq.Database, testReq.ID), nil)
	if err != nil {
		return err
	}

	return w.client.do(ctx, req, nil)
}

func webhooksAPIPath(org, db string) string {
	return path.Join("v1/organizations", org, "databases", db, "webhooks")
}

func webhookAPIPath(org, db, id string) string {
	return path.Join(webhooksAPIPath(org, db), id)
}

func webhookTestAPIPath(org, db, id string) string {
	return path.Join(webhookAPIPath(org, db, id), "test")
}
