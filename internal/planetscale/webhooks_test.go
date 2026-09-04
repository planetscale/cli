package planetscale

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestWebhooks_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/webhooks")

		out := `{
			"current_page": 1,
			"next_page": null,
			"next_page_url": null,
			"prev_page": null,
			"prev_page_url": null,
			"data": [{
				"id": "webhook-123",
				"url": "https://example.com/webhook",
				"secret": "secret-123",
				"authorization_header_configured": true,
				"enabled": true,
				"last_sent_result": "success",
				"last_sent_success": true,
				"last_sent_at": "2021-01-14T10:19:23.000Z",
				"created_at": "2021-01-14T10:19:23.000Z",
				"updated_at": "2021-01-14T10:19:23.000Z",
				"events": ["branch.ready", "deploy_request.opened"]
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	webhooks, err := client.Webhooks.List(ctx, &ListWebhooksRequest{
		Organization: testOrg,
		Database:     testDatabase,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(len(webhooks), qt.Equals, 1)
	c.Assert(webhooks[0].ID, qt.Equals, "webhook-123")
	c.Assert(webhooks[0].URL, qt.Equals, "https://example.com/webhook")
	c.Assert(webhooks[0].AuthorizationHeaderConfigured, qt.IsTrue)
	c.Assert(webhooks[0].Enabled, qt.IsTrue)
	c.Assert(webhooks[0].Events, qt.DeepEquals, []string{"branch.ready", "deploy_request.opened"})
}

func TestWebhooks_List_WithPagination(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "10")

		out := `{
			"current_page": 2,
			"next_page": null,
			"next_page_url": null,
			"prev_page": 1,
			"prev_page_url": "https://api.planetscale.com/v1/organizations/my-org/databases/planetscale-go-test-db/webhooks?page=1",
			"data": []
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	webhooks, err := client.Webhooks.List(ctx, &ListWebhooksRequest{
		Organization: testOrg,
		Database:     testDatabase,
	}, WithPage(2), WithPerPage(10))

	c.Assert(err, qt.IsNil)
	c.Assert(len(webhooks), qt.Equals, 0)
}

func TestWebhooks_Create(t *testing.T) {
	c := qt.New(t)

	wantBody := []byte("{\"url\":\"https://example.com/webhook\",\"authorization_header\":\"Bearer automation-token\",\"enabled\":true,\"events\":[\"branch.ready\"]}\n")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/webhooks")

		data, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(data, qt.DeepEquals, wantBody)

		out := `{
			"id": "webhook-123",
			"url": "https://example.com/webhook",
			"secret": "generated-secret",
			"enabled": true,
			"last_sent_result": "",
			"last_sent_success": false,
			"last_sent_at": "0001-01-01T00:00:00.000Z",
			"created_at": "2021-01-14T10:19:23.000Z",
			"updated_at": "2021-01-14T10:19:23.000Z",
			"events": ["branch.ready"]
		}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	enabled := true

	webhook, err := client.Webhooks.Create(ctx, &CreateWebhookRequest{
		Organization:        testOrg,
		Database:            testDatabase,
		URL:                 "https://example.com/webhook",
		AuthorizationHeader: "Bearer automation-token",
		Enabled:             &enabled,
		Events:              []string{"branch.ready"},
	})

	c.Assert(err, qt.IsNil)
	c.Assert(webhook.ID, qt.Equals, "webhook-123")
	c.Assert(webhook.URL, qt.Equals, "https://example.com/webhook")
	c.Assert(webhook.Secret, qt.Equals, "generated-secret")
	c.Assert(webhook.Enabled, qt.IsTrue)
	c.Assert(webhook.CreatedAt, qt.Equals, time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC))
}

func TestWebhooks_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/webhooks/webhook-123")

		out := `{
			"id": "webhook-123",
			"url": "https://example.com/webhook",
			"secret": "secret-123",
			"enabled": true,
			"last_sent_result": "success",
			"last_sent_success": true,
			"last_sent_at": "2021-01-14T10:19:23.000Z",
			"created_at": "2021-01-14T10:19:23.000Z",
			"updated_at": "2021-01-14T10:19:23.000Z",
			"events": ["branch.ready"]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	webhook, err := client.Webhooks.Get(ctx, &GetWebhookRequest{
		Organization: testOrg,
		Database:     testDatabase,
		ID:           "webhook-123",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(webhook.ID, qt.Equals, "webhook-123")
	c.Assert(webhook.URL, qt.Equals, "https://example.com/webhook")
	c.Assert(webhook.LastSentSuccess, qt.IsTrue)
}

func TestWebhooks_Update(t *testing.T) {
	c := qt.New(t)

	wantBody := []byte("{\"url\":\"https://example.com/new-webhook\",\"authorization_header\":\"Bearer automation-token\",\"enabled\":false}\n")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/webhooks/webhook-123")

		data, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(data, qt.DeepEquals, wantBody)

		out := `{
			"id": "webhook-123",
			"url": "https://example.com/new-webhook",
			"secret": "secret-123",
			"enabled": false,
			"last_sent_result": "success",
			"last_sent_success": true,
			"last_sent_at": "2021-01-14T10:19:23.000Z",
			"created_at": "2021-01-14T10:19:23.000Z",
			"updated_at": "2021-01-15T10:19:23.000Z",
			"events": ["branch.ready"]
		}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	newURL := "https://example.com/new-webhook"
	enabled := false
	authorizationHeader := "Bearer automation-token"

	webhook, err := client.Webhooks.Update(ctx, &UpdateWebhookRequest{
		Organization:        testOrg,
		Database:            testDatabase,
		ID:                  "webhook-123",
		URL:                 &newURL,
		AuthorizationHeader: &authorizationHeader,
		Enabled:             &enabled,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(webhook.ID, qt.Equals, "webhook-123")
	c.Assert(webhook.URL, qt.Equals, "https://example.com/new-webhook")
	c.Assert(webhook.Enabled, qt.IsFalse)
	c.Assert(webhook.UpdatedAt, qt.Equals, time.Date(2021, 1, 15, 10, 19, 23, 0, time.UTC))
}

func TestWebhooks_Update_ClearAuthorizationHeader(t *testing.T) {
	c := qt.New(t)

	wantBody := []byte("{\"authorization_header\":\"\"}\n")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/webhooks/webhook-123")

		data, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(data, qt.DeepEquals, wantBody)

		_, err = w.Write([]byte(`{"id":"webhook-123","url":"https://example.com/webhook"}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	empty := ""
	webhook, err := client.Webhooks.Update(context.Background(), &UpdateWebhookRequest{
		Organization:        testOrg,
		Database:            testDatabase,
		ID:                  "webhook-123",
		AuthorizationHeader: &empty,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(webhook.ID, qt.Equals, "webhook-123")
}

func TestWebhooks_Delete(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/webhooks/webhook-123")
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.Webhooks.Delete(ctx, &DeleteWebhookRequest{
		Organization: testOrg,
		Database:     testDatabase,
		ID:           "webhook-123",
	})

	c.Assert(err, qt.IsNil)
}

func TestWebhooks_Test(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/webhooks/webhook-123/test")
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.Webhooks.Test(ctx, &TestWebhookRequest{
		Organization: testOrg,
		Database:     testDatabase,
		ID:           "webhook-123",
	})

	c.Assert(err, qt.IsNil)
}
