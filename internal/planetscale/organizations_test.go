package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestOrganizations_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{
  "data": [
    {
      "id": "my-cool-org",
      "type": "organization",
	  "name": "my-cool-org",
	  "created_at": "2021-01-14T10:19:23.000Z",
	  "updated_at": "2021-01-14T10:19:23.000Z"
    }
  ]
}`

		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	orgs, err := client.Organizations.List(ctx)

	c.Assert(err, qt.IsNil)
	want := []*Organization{
		{
			Name:      "my-cool-org",
			CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
			UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		},
	}

	c.Assert(orgs, qt.DeepEquals, want)
}

func TestOrganizations_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{
      "id": "my-cool-org",
      "type": "organization",
      "name": "my-cool-org",
      "created_at": "2021-01-14T10:19:23.000Z",
      "updated_at": "2021-01-14T10:19:23.000Z"
}`

		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	org, err := client.Organizations.Get(ctx, &GetOrganizationRequest{
		Organization: "my-cool-org",
	})

	c.Assert(err, qt.IsNil)
	want := &Organization{
		Name:      "my-cool-org",
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(org, qt.DeepEquals, want)
}

func TestOrganizations_Update(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-cool-org")

		var body map[string]interface{}
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body, qt.DeepEquals, map[string]interface{}{
			"billing_email":         "billing@example.com",
			"idp_managed_roles":     false,
			"invoice_budget_amount": float64(2500),
		})

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"name": "my-cool-org",
			"billing_email": "billing@example.com",
			"idp_managed_roles": false,
			"invoice_budget_amount": 2500,
			"created_at": "2021-01-14T10:19:23.000Z",
			"updated_at": "2021-01-15T10:19:23.000Z"
		}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	org, err := client.Organizations.Update(context.Background(), &UpdateOrganizationRequest{
		Organization:        "my-cool-org",
		BillingEmail:        Pointer("billing@example.com"),
		IDPManagedRoles:     Pointer(false),
		InvoiceBudgetAmount: Pointer(float64(2500)),
	})
	c.Assert(err, qt.IsNil)
	c.Assert(org, qt.DeepEquals, &Organization{
		Name:                "my-cool-org",
		BillingEmail:        "billing@example.com",
		IDPManagedRoles:     false,
		InvoiceBudgetAmount: 2500,
		CreatedAt:           time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt:           time.Date(2021, time.January, 15, 10, 19, 23, 0, time.UTC),
	})
}

func TestOrganizations_ListRegions(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{
"data": [
		{
			"id": "my-cool-org",
			"type": "Region",
			"slug": "us-east",
			"display_name": "US East",
			"enabled": true
		}
	]
}`

		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	orgs, err := client.Organizations.ListRegions(ctx, &ListOrganizationRegionsRequest{
		Organization: "my-cool-org",
	})

	c.Assert(err, qt.IsNil)
	want := []*Region{
		{
			Name:    "US East",
			Slug:    "us-east",
			Enabled: true,
		},
	}

	c.Assert(orgs, qt.DeepEquals, want)
}

func TestOrganizations_ListClusterSKUs(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-cool-org/cluster-size-skus")
		out := `[
		{
			"name": "PS_10",
			"type": "ClusterSizeSku",
			"display_name": "PS-10",
			"cpu": "1/8",
			"provider_instance_type": null,
			"storage": null,
			"ram": 1,
			"metal": true,
			"enabled": true,
			"provider": null,
			"rate": null,
			"replica_rate": null,
			"default_vtgate": "VTG_5",
			"default_vtgate_rate": null,
			"sort_order": 1
		}
	]`

		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	orgs, err := client.Organizations.ListClusterSKUs(ctx, &ListOrganizationClusterSKUsRequest{
		Organization: "my-cool-org",
	})

	c.Assert(err, qt.IsNil)
	want := []*ClusterSKU{
		{
			Name:          "PS_10",
			DisplayName:   "PS-10",
			CPU:           "1/8",
			Memory:        1,
			Enabled:       true,
			DefaultVTGate: "VTG_5",
			SortOrder:     1,
			Metal:         true,
		},
	}

	c.Assert(orgs, qt.DeepEquals, want)
}

func TestOrganizations_ListClusterSKUsWithRates(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-cool-org/cluster-size-skus?rates=true")
		out := `[
		{
			"name": "PS_10",
			"type": "ClusterSizeSku",
			"display_name": "PS-10",
			"cpu": "1/8",
			"provider_instance_type": null,
			"storage": 100,
			"ram": 1,
			"sort_order": 1,
			"enabled": true,
			"provider": null,
			"rate": 39,
			"replica_rate": 13,
			"default_vtgate": "VTG_5",
			"default_vtgate_rate": null
		}
	]`

		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	orgs, err := client.Organizations.ListClusterSKUs(ctx, &ListOrganizationClusterSKUsRequest{
		Organization: "my-cool-org",
	}, WithRates())

	c.Assert(err, qt.IsNil)
	want := []*ClusterSKU{
		{
			Name:          "PS_10",
			DisplayName:   "PS-10",
			CPU:           "1/8",
			Memory:        1,
			Enabled:       true,
			Storage:       Pointer[int64](100),
			Rate:          Pointer[int64](39),
			ReplicaRate:   Pointer[int64](13),
			DefaultVTGate: "VTG_5",
			SortOrder:     1,
		},
	}

	c.Assert(orgs, qt.DeepEquals, want)
}

func TestOrganizations_ListClusterSKUsWithPostgreSQL(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-cool-org/cluster-size-skus?postgresql=true")
		out := `[
		{
			"name": "PS_10_AWS_ARM",
			"type": "ClusterSizeSku",
			"display_name": "PS-10-AWS-ARM",
			"cpu": "1/8",
			"provider_instance_type": null,
			"storage": null,
			"ram": 1,
			"sort_order": 1,
			"enabled": true,
			"provider": "aws",
			"rate": 34,
			"replica_rate": null,
			"default_vtgate": null,
			"default_vtgate_rate": null
		}
	]`

		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	orgs, err := client.Organizations.ListClusterSKUs(ctx, &ListOrganizationClusterSKUsRequest{
		Organization: "my-cool-org",
	}, WithPostgreSQL())

	c.Assert(err, qt.IsNil)
	want := []*ClusterSKU{
		{
			Name:        "PS_10_AWS_ARM",
			DisplayName: "PS-10-AWS-ARM",
			CPU:         "1/8",
			Memory:      1,
			Enabled:     true,
			Provider:    Pointer("aws"),
			Rate:        Pointer[int64](34),
			SortOrder:   1,
		},
	}

	c.Assert(orgs, qt.DeepEquals, want)
}
