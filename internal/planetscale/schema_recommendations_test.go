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

func TestSchemaRecommendations_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/schema-recommendations")

		out := `{
			"current_page": 1,
			"next_page": null,
			"next_page_url": null,
			"prev_page": null,
			"prev_page_url": null,
			"data": [{
				"id": "recommendation-123",
				"html_url": "https://app.planetscale.com/my-org/planetscale-go-test-db/insights/recommendations/1",
				"title": "Add index on users.email",
				"table_name": "users",
				"keyspace": "main",
				"ddl_statement": "ALTER TABLE users ADD INDEX idx_email (email)",
				"number": 1,
				"state": "open",
				"recommendation_type": "index",
				"created_at": "2021-01-14T10:19:23.000Z",
				"updated_at": "2021-01-14T10:19:23.000Z",
				"applied_at": null,
				"dismissed_at": null,
				"closed_by_deploy_request": {
					"id": "",
					"branch_id": "",
					"number": 0
				},
				"dismissed_by": {
					"id": "",
					"display_name": ""
				}
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	recommendations, err := client.SchemaRecommendations.List(ctx, &ListSchemaRecommendationsRequest{
		Organization: testOrg,
		Database:     testDatabase,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(len(recommendations), qt.Equals, 1)
	c.Assert(recommendations[0].ID, qt.Equals, "recommendation-123")
	c.Assert(recommendations[0].Title, qt.Equals, "Add index on users.email")
	c.Assert(recommendations[0].Table, qt.Equals, "users")
	c.Assert(recommendations[0].Keyspace, qt.Equals, "main")
	c.Assert(recommendations[0].State, qt.Equals, "open")
	c.Assert(recommendations[0].RecommendationType, qt.Equals, "index")
	c.Assert(recommendations[0].Number, qt.Equals, 1)
}

func TestSchemaRecommendations_List_WithPagination(t *testing.T) {
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
			"prev_page_url": "https://api.planetscale.com/v1/organizations/my-org/databases/planetscale-go-test-db/schema-recommendations?page=1",
			"data": []
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	recommendations, err := client.SchemaRecommendations.List(ctx, &ListSchemaRecommendationsRequest{
		Organization: testOrg,
		Database:     testDatabase,
	}, WithPage(2), WithPerPage(10))

	c.Assert(err, qt.IsNil)
	c.Assert(len(recommendations), qt.Equals, 0)
}

func TestSchemaRecommendations_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/schema-recommendations/1")

		out := `{
			"id": "recommendation-123",
			"html_url": "https://app.planetscale.com/my-org/planetscale-go-test-db/insights/recommendations/1",
			"title": "Add index on users.email",
			"table_name": "users",
			"keyspace": "main",
			"ddl_statement": "ALTER TABLE users ADD INDEX idx_email (email)",
			"number": 1,
			"state": "open",
			"recommendation_type": "index",
			"created_at": "2021-01-14T10:19:23.000Z",
			"updated_at": "2021-01-14T10:19:23.000Z"
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	recommendation, err := client.SchemaRecommendations.Get(ctx, &GetSchemaRecommendationRequest{
		Organization: testOrg,
		Database:     testDatabase,
		ID:           "1",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(recommendation.ID, qt.Equals, "recommendation-123")
	c.Assert(recommendation.Number, qt.Equals, 1)
	c.Assert(recommendation.Title, qt.Equals, "Add index on users.email")
	c.Assert(recommendation.DDLStatement, qt.Equals, "ALTER TABLE users ADD INDEX idx_email (email)")
	c.Assert(recommendation.State, qt.Equals, "open")
}

func TestSchemaRecommendations_Dismiss(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/schema-recommendations/1/dismiss")

		var body map[string]any
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body["reason"], qt.Equals, "not applicable")

		out := `{
			"id": "recommendation-123",
			"html_url": "https://app.planetscale.com/my-org/planetscale-go-test-db/insights/recommendations/1",
			"title": "Add index on users.email",
			"table_name": "users",
			"keyspace": "main",
			"ddl_statement": "ALTER TABLE users ADD INDEX idx_email (email)",
			"number": 1,
			"state": "dismissed",
			"recommendation_type": "index",
			"created_at": "2021-01-14T10:19:23.000Z",
			"updated_at": "2021-01-15T10:19:23.000Z",
			"applied_at": null,
			"dismissed_at": "2021-01-15T10:19:23.000Z",
			"dismissed_by": {
				"id": "user-123",
				"display_name": "Test User"
			}
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	recommendation, err := client.SchemaRecommendations.Dismiss(ctx, &DismissSchemaRecommendationRequest{
		Organization: testOrg,
		Database:     testDatabase,
		ID:           "1",
		Reason:       "not applicable",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(recommendation.ID, qt.Equals, "recommendation-123")
	c.Assert(recommendation.State, qt.Equals, "dismissed")
	c.Assert(recommendation.DismissedAt, qt.IsNotNil)
	c.Assert(recommendation.DismissedAt.Format(time.RFC3339Nano), qt.Equals, "2021-01-15T10:19:23Z")
	c.Assert(recommendation.DismissedBy.ID, qt.Equals, "user-123")
	c.Assert(recommendation.DismissedBy.Name, qt.Equals, "Test User")
}
