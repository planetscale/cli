package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
