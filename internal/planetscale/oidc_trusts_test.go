package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestOIDCTrusts_ListProviders(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/oidc-trust-providers")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data": [{
			"name": "github",
			"display_name": "GitHub Actions",
			"issuer": "https://token.actions.githubusercontent.com",
			"required_claims": ["repository", "ref"]
		}]}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	providers, err := client.OIDCTrusts.ListProviders(context.Background(), &ListOIDCTrustProvidersRequest{Organization: testOrg})
	c.Assert(err, qt.IsNil)
	c.Assert(providers, qt.DeepEquals, []*OIDCTrustProvider{{
		Name:           "github",
		DisplayName:    "GitHub Actions",
		Issuer:         "https://token.actions.githubusercontent.com",
		RequiredClaims: []string{"repository", "ref"},
	}})
}

func TestOIDCTrusts_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/oidc-trusts/trust-1")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{
			"id": "trust-1",
			"display_name": "CI",
			"provider": "github",
			"audience": "my-org",
			"max_ttl_seconds": 3600,
			"created_at": "2025-01-15T10:30:00Z",
			"updated_at": "2025-01-16T10:30:00Z",
			"conditions": [{"id": "c1", "claim": "repository", "value": "acme/app"}],
			"accesses": [{"id": "a1", "access": "connect_branch", "resource_name": "mydb", "resource_id": "db1", "resource_type": "Database"}],
			"bulk_database_accesses": ["read_branch"]
		}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	trust, err := client.OIDCTrusts.Get(context.Background(), &GetOIDCTrustRequest{Organization: testOrg, ID: "trust-1"})
	c.Assert(err, qt.IsNil)
	c.Assert(trust, qt.DeepEquals, &OIDCTrust{
		ID:            "trust-1",
		DisplayName:   "CI",
		Provider:      "github",
		Audience:      "my-org",
		MaxTTLSeconds: 3600,
		CreatedAt:     time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2025, 1, 16, 10, 30, 0, 0, time.UTC),
		Conditions:    []OIDCTrustCondition{{ID: "c1", Claim: "repository", Value: "acme/app"}},
		Accesses: []OIDCTrustAccess{{
			ID: "a1", Access: "connect_branch", ResourceName: "mydb", ResourceID: "db1", ResourceType: "Database",
		}},
		BulkDatabaseAccesses: []string{"read_branch"},
	})
}
