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

func TestServiceTokens_Create(t *testing.T) {
	c := qt.New(t)

	wantBody := []byte("{\"name\":\"my-token\"}\n")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		data, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(data, qt.DeepEquals, wantBody)

		out := `{"id":"test-id","type":"ServiceToken","token":"d2980bbd91a4ab878601ef0573a7af7b1b15e705","name":"my-token","created_at":"2021-01-14T10:19:23.000Z","last_used_at":"2021-01-15T12:30:00.000Z"}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	tokenName := "my-token"

	snapshot, err := client.ServiceTokens.Create(ctx, &CreateServiceTokenRequest{
		Organization: testOrg,
		Name:         &tokenName,
	})
	lastUsedAt := time.Date(2021, 1, 15, 12, 30, 0, 0, time.UTC)
	want := &ServiceToken{
		ID:         "test-id",
		Type:       "ServiceToken",
		Token:      "d2980bbd91a4ab878601ef0573a7af7b1b15e705",
		Name:       &tokenName,
		CreatedAt:  time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		LastUsedAt: &lastUsedAt,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(snapshot, qt.DeepEquals, want)
}

func TestServiceTokens_CreateWithTTL(t *testing.T) {
	c := qt.New(t)

	wantBody := []byte("{\"name\":\"my-token\",\"ttl\":3600}\n")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		data, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(data, qt.DeepEquals, wantBody)

		out := `{"id":"test-id","type":"ServiceToken","token":"d2980bbd91a4ab878601ef0573a7af7b1b15e705","name":"my-token","created_at":"2021-01-14T10:19:23.000Z","last_used_at":"2021-01-15T12:30:00.000Z","expires_at":"2021-01-14T11:19:23.000Z"}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	tokenName := "my-token"
	ttl := 3600

	snapshot, err := client.ServiceTokens.Create(ctx, &CreateServiceTokenRequest{
		Organization: testOrg,
		Name:         &tokenName,
		TTL:          &ttl,
	})
	lastUsedAt := time.Date(2021, 1, 15, 12, 30, 0, 0, time.UTC)
	expiresAt := time.Date(2021, 1, 14, 11, 19, 23, 0, time.UTC)
	want := &ServiceToken{
		ID:         "test-id",
		Type:       "ServiceToken",
		Token:      "d2980bbd91a4ab878601ef0573a7af7b1b15e705",
		Name:       &tokenName,
		CreatedAt:  time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		LastUsedAt: &lastUsedAt,
		ExpiresAt:  &expiresAt,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(snapshot, qt.DeepEquals, want)
}

func TestServiceTokens_Get(t *testing.T) {
	c := qt.New(t)

	wantURL := "/v1/organizations/my-org/service-tokens/1234"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, wantURL)

		out := `{"id":"1234","name":"my-token","token":null,"created_at":"2021-01-14T10:19:23.000Z","last_used_at":"2021-01-15T12:30:00.000Z","service_token_accesses":[{"access":"read_branch"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	token, err := client.ServiceTokens.Get(context.Background(), &GetServiceTokenRequest{
		Organization: testOrg,
		ID:           "1234",
	})

	tokenName := "my-token"
	lastUsedAt := time.Date(2021, 1, 15, 12, 30, 0, 0, time.UTC)
	want := &ServiceToken{
		ID:         "1234",
		Name:       &tokenName,
		CreatedAt:  time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		LastUsedAt: &lastUsedAt,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(token, qt.DeepEquals, want)
}

func TestServiceTokens_ListGrants(t *testing.T) {
	c := qt.New(t)

	out := `{"type":"list","current_page":1,"next_page":null,"next_page_url":null,"prev_page":null,"prev_page_url":null,"data":[{"id":"qbphfi83nxti","type":"ServiceTokenGrant","resource_name":"planetscale","resource_type":"Database","resource_id":"qbphfi83nxti","accesses":[{"access": "read_branch", "description": "Read database branch"}]}]}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	grants, err := client.ServiceTokens.ListGrants(ctx, &ListServiceTokenGrantsRequest{
		Organization: testOrg,
		ID:           "1234",
	})

	want := []*ServiceTokenGrant{
		{
			ID:           "qbphfi83nxti",
			ResourceName: "planetscale",
			ResourceType: "Database",
			ResourceID:   "qbphfi83nxti",
			Accesses:     []*ServiceTokenGrantAccess{{Access: "read_branch", Description: "Read database branch"}},
		},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(grants, qt.DeepEquals, want)
}

func TestServiceTokens_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"type":"list","next_page":null,"prev_page":null,"data":[{"id":"txhc257pxjuc","type":"ServiceToken","token":null,"name":"list-token","created_at":"2021-01-14T10:19:23.000Z","last_used_at":null}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	snapshot, err := client.ServiceTokens.List(ctx, &ListServiceTokensRequest{
		Organization: testOrg,
	})
	tokenName := "list-token"
	want := []*ServiceToken{
		{
			ID:        "txhc257pxjuc",
			Type:      "ServiceToken",
			Name:      &tokenName,
			CreatedAt: time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(snapshot, qt.DeepEquals, want)
}

func TestServiceTokens_Delete(t *testing.T) {
	c := qt.New(t)

	wantURL := "/v1/organizations/my-org/service-tokens/1234"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.URL.String(), qt.DeepEquals, wantURL)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.ServiceTokens.Delete(ctx, &DeleteServiceTokenRequest{
		Organization: testOrg,
		ID:           "1234",
	})

	c.Assert(err, qt.IsNil)
}

func TestServiceTokens_GetAccess(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"type":"list","next_page":null,"prev_page":null,"data":[{"id":"hjqui654yu71","type":"DatabaseAccess","resource":{"id":"1lbjwnp48b6r","type":"Database","name":"hidden-river-4209","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"},"access":"read_comment"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	snapshot, err := client.ServiceTokens.GetAccess(ctx, &GetServiceTokenAccessRequest{
		Organization: testOrg,
		ID:           "1234",
	})
	want := []*ServiceTokenAccess{
		{
			ID:     "hjqui654yu71",
			Access: "read_comment",
			Type:   "DatabaseAccess",
			Resource: ServiceTokenResource{
				ID:   "1lbjwnp48b6r",
				Name: "hidden-river-4209",
				Type: "Database",
			},
		},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(snapshot, qt.DeepEquals, want)
}

func TestServiceTokens_AddAccess(t *testing.T) {
	c := qt.New(t)

	wantBody := []byte("{\"database\":\"hidden-river-4209\",\"access\":[\"read_comment\"]}\n")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		data, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(data, qt.DeepEquals, wantBody)

		out := `{"type":"list","next_page":null,"prev_page":null,"data":[{"id":"hjqui654yu71","type":"DatabaseAccess","resource":{"id":"1lbjwnp48b6r","type":"Database","name":"hidden-river-4209","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"},"access":"read_comment"}]}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	snapshot, err := client.ServiceTokens.AddAccess(ctx, &AddServiceTokenAccessRequest{
		Organization: testOrg,
		ID:           "1234",
		Database:     "hidden-river-4209",
		Accesses:     []string{"read_comment"},
	})
	want := []*ServiceTokenAccess{
		{
			ID:     "hjqui654yu71",
			Access: "read_comment",
			Type:   "DatabaseAccess",
			Resource: ServiceTokenResource{
				ID:   "1lbjwnp48b6r",
				Name: "hidden-river-4209",
				Type: "Database",
			},
		},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(snapshot, qt.DeepEquals, want)
}

func TestServiceTokens_DeleteAccess(t *testing.T) {
	c := qt.New(t)

	wantBody := []byte("{\"database\":\"hidden-river-4209\",\"access\":[\"read_comment\"]}\n")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		data, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(data, qt.DeepEquals, wantBody)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.ServiceTokens.DeleteAccess(ctx, &DeleteServiceTokenAccessRequest{
		Organization: testOrg,
		ID:           "1234",
		Database:     "hidden-river-4209",
		Accesses:     []string{"read_comment"},
	})

	c.Assert(err, qt.IsNil)
}
