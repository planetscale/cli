package mock

import (
	"context"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type WebhooksService struct {
	ListFn        func(context.Context, *ps.ListWebhooksRequest, ...ps.ListOption) ([]*ps.Webhook, error)
	ListFnInvoked bool

	CreateFn        func(context.Context, *ps.CreateWebhookRequest) (*ps.Webhook, error)
	CreateFnInvoked bool

	GetFn        func(context.Context, *ps.GetWebhookRequest) (*ps.Webhook, error)
	GetFnInvoked bool

	UpdateFn        func(context.Context, *ps.UpdateWebhookRequest) (*ps.Webhook, error)
	UpdateFnInvoked bool

	DeleteFn        func(context.Context, *ps.DeleteWebhookRequest) error
	DeleteFnInvoked bool

	TestFn        func(context.Context, *ps.TestWebhookRequest) error
	TestFnInvoked bool
}

func (w *WebhooksService) List(ctx context.Context, req *ps.ListWebhooksRequest, opts ...ps.ListOption) ([]*ps.Webhook, error) {
	w.ListFnInvoked = true
	return w.ListFn(ctx, req, opts...)
}

func (w *WebhooksService) Create(ctx context.Context, req *ps.CreateWebhookRequest) (*ps.Webhook, error) {
	w.CreateFnInvoked = true
	return w.CreateFn(ctx, req)
}

func (w *WebhooksService) Get(ctx context.Context, req *ps.GetWebhookRequest) (*ps.Webhook, error) {
	w.GetFnInvoked = true
	return w.GetFn(ctx, req)
}

func (w *WebhooksService) Update(ctx context.Context, req *ps.UpdateWebhookRequest) (*ps.Webhook, error) {
	w.UpdateFnInvoked = true
	return w.UpdateFn(ctx, req)
}

func (w *WebhooksService) Delete(ctx context.Context, req *ps.DeleteWebhookRequest) error {
	w.DeleteFnInvoked = true
	return w.DeleteFn(ctx, req)
}

func (w *WebhooksService) Test(ctx context.Context, req *ps.TestWebhookRequest) error {
	w.TestFnInvoked = true
	return w.TestFn(ctx, req)
}

