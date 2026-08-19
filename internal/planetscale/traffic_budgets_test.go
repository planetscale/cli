package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestTrafficBudgets_ListWithFingerprint(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/planetscale-go-test-db-branch/traffic/budgets")
		c.Assert(r.URL.Query().Get("fingerprint"), qt.Equals, "abc123")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":[]}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	budgets, err := client.TrafficBudgets.List(context.Background(), &ListTrafficBudgetsRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       testBranch,
		Fingerprint:  "abc123",
	})

	c.Assert(err, qt.IsNil)
	c.Assert(budgets, qt.HasLen, 0)
}

func TestTrafficBudgets_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/planetscale-go-test-db-branch/traffic/budgets")

		w.WriteHeader(200)
		out := `{"data":[
			{
				"id":"qok87ki4xlau",
				"type":"TrafficBudget",
				"name":"my-budget",
				"mode":"warn",
				"capacity":0,
				"rate":1,
				"burst":null,
				"concurrency":null,
				"created_at":"2026-03-20T15:18:08.540Z",
				"updated_at":"2026-03-20T15:21:36.068Z",
				"actor":{"id":"v1bxjxtt9c13","type":"User","display_name":"Alice"},
				"rules":[
					{
						"id":"ixrnbyuznjza",
						"type":"TrafficRule",
						"kind":"match",
						"tags":[
							{"type":"TrafficRuleTag","key_id":"Bremote_address","key":"remote_address","value":"10.0.0.8/10","source":"system"}
						],
						"fingerprint":null,
						"keyspace":null,
						"created_at":"2026-03-20T16:05:58.438Z",
						"updated_at":"2026-03-20T16:05:58.438Z",
						"actor":{"id":"v1bxjxtt9c13","type":"User","display_name":"Alice"}
					}
				]
			},
			{
				"id":"0h41b131ivqd",
				"type":"TrafficBudget",
				"name":"IP Range",
				"mode":"warn",
				"capacity":200,
				"rate":50,
				"burst":100,
				"concurrency":100,
				"created_at":"2026-03-20T15:58:23.153Z",
				"updated_at":"2026-03-20T15:58:23.153Z",
				"actor":{"id":"v1bxjxtt9c13","type":"User","display_name":"Alice"},
				"rules":[
					{
						"id":"96j6gkf2sbv8",
						"type":"TrafficRule",
						"kind":"match",
						"tags":[
							{"type":"TrafficRuleTag","key_id":"Bremote_address","key":"remote_address","value":"10.0.0.0/8","source":"system"}
						],
						"fingerprint":null,
						"keyspace":null,
						"created_at":"2026-03-20T15:58:23.177Z",
						"updated_at":"2026-03-20T15:58:23.177Z",
						"actor":{"id":"v1bxjxtt9c13","type":"User","display_name":"Alice"}
					}
				]
			}
		]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	budgets, err := client.TrafficBudgets.List(ctx, &ListTrafficBudgetsRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       testBranch,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(budgets, qt.HasLen, 2)

	actor := Actor{ID: "v1bxjxtt9c13", Type: "User", Name: "Alice"}

	want0 := &TrafficBudget{
		ID:       "qok87ki4xlau",
		Type:     "TrafficBudget",
		Name:     "my-budget",
		Mode:     "warn",
		Capacity: new(0),
		Rate:     new(1),
		Actor:    actor,
		Rules: []TrafficRule{
			{
				ID:   "ixrnbyuznjza",
				Type: "TrafficRule",
				Kind: "match",
				Tags: []TrafficRuleTag{
					{Type: "TrafficRuleTag", KeyID: "Bremote_address", Key: "remote_address", Value: "10.0.0.8/10", Source: "system"},
				},
				Actor:     actor,
				CreatedAt: time.Date(2026, time.March, 20, 16, 5, 58, 438000000, time.UTC),
				UpdatedAt: time.Date(2026, time.March, 20, 16, 5, 58, 438000000, time.UTC),
			},
		},
		CreatedAt: time.Date(2026, time.March, 20, 15, 18, 8, 540000000, time.UTC),
		UpdatedAt: time.Date(2026, time.March, 20, 15, 21, 36, 68000000, time.UTC),
	}

	want1 := &TrafficBudget{
		ID:          "0h41b131ivqd",
		Type:        "TrafficBudget",
		Name:        "IP Range",
		Mode:        "warn",
		Capacity:    new(200),
		Rate:        new(50),
		Burst:       new(100),
		Concurrency: new(100),
		Actor:       actor,
		Rules: []TrafficRule{
			{
				ID:   "96j6gkf2sbv8",
				Type: "TrafficRule",
				Kind: "match",
				Tags: []TrafficRuleTag{
					{Type: "TrafficRuleTag", KeyID: "Bremote_address", Key: "remote_address", Value: "10.0.0.0/8", Source: "system"},
				},
				Actor:     actor,
				CreatedAt: time.Date(2026, time.March, 20, 15, 58, 23, 177000000, time.UTC),
				UpdatedAt: time.Date(2026, time.March, 20, 15, 58, 23, 177000000, time.UTC),
			},
		},
		CreatedAt: time.Date(2026, time.March, 20, 15, 58, 23, 153000000, time.UTC),
		UpdatedAt: time.Date(2026, time.March, 20, 15, 58, 23, 153000000, time.UTC),
	}

	c.Assert(budgets[0], qt.DeepEquals, want0)
	c.Assert(budgets[1], qt.DeepEquals, want1)
}

func TestTrafficBudgets_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/planetscale-go-test-db-branch/traffic/budgets/qok87ki4xlau")

		w.WriteHeader(200)
		out := `{
			"id":"qok87ki4xlau",
			"type":"TrafficBudget",
			"name":"my-budget",
			"mode":"warn",
			"capacity":0,
			"rate":1,
			"burst":null,
			"concurrency":null,
			"created_at":"2026-03-20T15:18:08.540Z",
			"updated_at":"2026-03-20T15:21:36.068Z",
			"actor":{"id":"v1bxjxtt9c13","type":"User","display_name":"Alice"},
			"rules":[
				{
					"id":"ixrnbyuznjza",
					"type":"TrafficRule",
					"kind":"match",
					"tags":[
						{"type":"TrafficRuleTag","key_id":"Bremote_address","key":"remote_address","value":"10.0.0.8/10","source":"system"},
						{"type":"TrafficRuleTag","key_id":"Squery","key":"query","value":"a_query","source":"sql"}
					],
					"fingerprint":null,
					"keyspace":null,
					"created_at":"2026-03-20T16:05:58.438Z",
					"updated_at":"2026-03-20T16:05:58.438Z",
					"actor":{"id":"v1bxjxtt9c13","type":"User","display_name":"Alice"}
				}
			]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	budget, err := client.TrafficBudgets.Get(ctx, &GetTrafficBudgetRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       testBranch,
		BudgetID:     "qok87ki4xlau",
	})

	actor := Actor{ID: "v1bxjxtt9c13", Type: "User", Name: "Alice"}

	want := &TrafficBudget{
		ID:       "qok87ki4xlau",
		Type:     "TrafficBudget",
		Name:     "my-budget",
		Mode:     "warn",
		Capacity: new(0),
		Rate:     new(1),
		Actor:    actor,
		Rules: []TrafficRule{
			{
				ID:   "ixrnbyuznjza",
				Type: "TrafficRule",
				Kind: "match",
				Tags: []TrafficRuleTag{
					{Type: "TrafficRuleTag", KeyID: "Bremote_address", Key: "remote_address", Value: "10.0.0.8/10", Source: "system"},
					{Type: "TrafficRuleTag", KeyID: "Squery", Key: "query", Value: "a_query", Source: "sql"},
				},
				Actor:     actor,
				CreatedAt: time.Date(2026, time.March, 20, 16, 5, 58, 438000000, time.UTC),
				UpdatedAt: time.Date(2026, time.March, 20, 16, 5, 58, 438000000, time.UTC),
			},
		},
		CreatedAt: time.Date(2026, time.March, 20, 15, 18, 8, 540000000, time.UTC),
		UpdatedAt: time.Date(2026, time.March, 20, 15, 21, 36, 68000000, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(budget, qt.DeepEquals, want)
}

func TestTrafficBudgets_Create(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/planetscale-go-test-db-branch/traffic/budgets")

		w.WriteHeader(200)
		out := `{
			"id":"3ohd8d28icus",
			"type":"TrafficBudget",
			"name":"query",
			"mode":"warn",
			"capacity":200,
			"rate":50,
			"burst":100,
			"concurrency":100,
			"created_at":"2026-03-20T16:05:06.866Z",
			"updated_at":"2026-03-20T16:05:06.866Z",
			"actor":{"id":"v1bxjxtt9c13","type":"User","display_name":"Alice"},
			"rules":[
				{
					"id":"rv8lbt2t02sz",
					"type":"TrafficRule",
					"kind":"match",
					"tags":[
						{"type":"TrafficRuleTag","key_id":"Bremote_address","key":"remote_address","value":"192.168.1.1","source":"system"},
						{"type":"TrafficRuleTag","key_id":"Squery","key":"query","value":"a_query","source":"sql"}
					],
					"fingerprint":null,
					"keyspace":null,
					"created_at":"2026-03-20T16:05:06.894Z",
					"updated_at":"2026-03-20T16:05:06.894Z",
					"actor":{"id":"v1bxjxtt9c13","type":"User","display_name":"Alice"}
				}
			]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	budget, err := client.TrafficBudgets.Create(ctx, &CreateTrafficBudgetRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       testBranch,
		Name:         "query",
		Mode:         "warn",
		Capacity:     new(200),
		Rate:         new(50),
		Burst:        new(100),
		Concurrency:  new(100),
		Rules: &[]CreateTrafficBudgetRuleRequest{
			{
				Kind: "match",
				Tags: &[]TrafficRuleTag{
					{Key: "remote_address", Value: "192.168.1.1", Source: "system"},
					{Key: "query", Value: "a_query", Source: "sql"},
				},
			},
		},
	})

	actor := Actor{ID: "v1bxjxtt9c13", Type: "User", Name: "Alice"}

	want := &TrafficBudget{
		ID:          "3ohd8d28icus",
		Type:        "TrafficBudget",
		Name:        "query",
		Mode:        "warn",
		Capacity:    new(200),
		Rate:        new(50),
		Burst:       new(100),
		Concurrency: new(100),
		Actor:       actor,
		Rules: []TrafficRule{
			{
				ID:   "rv8lbt2t02sz",
				Type: "TrafficRule",
				Kind: "match",
				Tags: []TrafficRuleTag{
					{Type: "TrafficRuleTag", KeyID: "Bremote_address", Key: "remote_address", Value: "192.168.1.1", Source: "system"},
					{Type: "TrafficRuleTag", KeyID: "Squery", Key: "query", Value: "a_query", Source: "sql"},
				},
				Actor:     actor,
				CreatedAt: time.Date(2026, time.March, 20, 16, 5, 6, 894000000, time.UTC),
				UpdatedAt: time.Date(2026, time.March, 20, 16, 5, 6, 894000000, time.UTC),
			},
		},
		CreatedAt: time.Date(2026, time.March, 20, 16, 5, 6, 866000000, time.UTC),
		UpdatedAt: time.Date(2026, time.March, 20, 16, 5, 6, 866000000, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(budget, qt.DeepEquals, want)
}

func TestTrafficBudgets_Update(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/planetscale-go-test-db-branch/traffic/budgets/qok87ki4xlau")

		w.WriteHeader(200)
		out := `{
			"id":"qok87ki4xlau",
			"type":"TrafficBudget",
			"name":"my-budget updated",
			"mode":"enforce",
			"capacity":500,
			"rate":100,
			"burst":200,
			"concurrency":50,
			"created_at":"2026-03-20T15:18:08.540Z",
			"updated_at":"2026-03-20T17:30:00.000Z",
			"actor":{"id":"v1bxjxtt9c13","type":"User","display_name":"Alice"},
			"rules":[]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	name := "my-budget updated"
	mode := "enforce"
	capacity := 500
	rate := 100
	burst := 200
	concurrency := 50

	budget, err := client.TrafficBudgets.Update(ctx, &UpdateTrafficBudgetRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       testBranch,
		BudgetID:     "qok87ki4xlau",
		Name:         &name,
		Mode:         &mode,
		Capacity:     &capacity,
		Rate:         &rate,
		Burst:        &burst,
		Concurrency:  &concurrency,
	})

	want := &TrafficBudget{
		ID:          "qok87ki4xlau",
		Type:        "TrafficBudget",
		Name:        "my-budget updated",
		Mode:        "enforce",
		Capacity:    new(500),
		Rate:        new(100),
		Burst:       new(200),
		Concurrency: new(50),
		Actor:       Actor{ID: "v1bxjxtt9c13", Type: "User", Name: "Alice"},
		Rules:       []TrafficRule{},
		CreatedAt:   time.Date(2026, time.March, 20, 15, 18, 8, 540000000, time.UTC),
		UpdatedAt:   time.Date(2026, time.March, 20, 17, 30, 0, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(budget, qt.DeepEquals, want)
}

func TestTrafficBudgets_Delete(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.String(), qt.Equals, "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/planetscale-go-test-db-branch/traffic/budgets/qok87ki4xlau")

		w.WriteHeader(http.StatusNoContent)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.TrafficBudgets.Delete(ctx, &DeleteTrafficBudgetRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       testBranch,
		BudgetID:     "qok87ki4xlau",
	})

	c.Assert(err, qt.IsNil)
}
