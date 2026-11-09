package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestOIDCTrusts_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/oidc-trusts/trust-123")
		_, err := w.Write([]byte(`{
			"id": "trust-123",
			"display_name": "CI",
			"provider": "github",
			"audience": "https://example.com",
			"max_ttl_seconds": 3600,
			"created_at": "2021-01-14T10:19:23.000Z",
			"updated_at": "2021-01-14T10:19:23.000Z",
			"conditions": [{"id": "cond-1", "claim": "repository", "value": "acme/app"}],
			"accesses": [{"id": "acc-1", "access": "read_branch", "resource_name": "mydb", "resource_id": "db-1", "resource_type": "Database"}],
			"bulk_database_accesses": ["read_branch"]
		}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	trust, err := client.OIDCTrusts.Get(context.Background(), &GetOIDCTrustRequest{Organization: testOrg, ID: "trust-123"})
	c.Assert(err, qt.IsNil)

	ts0 := time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC)
	c.Assert(trust, qt.DeepEquals, &OIDCTrust{
		ID:                   "trust-123",
		DisplayName:          "CI",
		Provider:             "github",
		Audience:             "https://example.com",
		MaxTTLSeconds:        3600,
		CreatedAt:            ts0,
		UpdatedAt:            ts0,
		Conditions:           []OIDCTrustCondition{{ID: "cond-1", Claim: "repository", Value: "acme/app"}},
		Accesses:             []OIDCTrustAccess{{ID: "acc-1", Access: "read_branch", ResourceName: "mydb", ResourceID: "db-1", ResourceType: "Database"}},
		BulkDatabaseAccesses: []string{"read_branch"},
	})
}
