package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestNekiRoutersService(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/routers":
			_, _ = w.Write([]byte(`[{
				"type":"neki_router",
				"name":"default",
				"default":true,
				"sku":{"name":"NKR_5","display_name":"NKR-5","cpu":"1","ram":4294967296,"sort_order":1},
				"replicas_per_cell":1,
				"autoscaling":false,
				"max_replicas_per_cell":null,
				"target_cpu_utilization":null,
				"state":"ready",
				"created_at":"2026-08-04T12:00:00Z",
				"updated_at":"2026-08-04T12:00:00Z"
			}]`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/routers/analytics":
			_, _ = w.Write([]byte(`{
				"type":"neki_router",
				"name":"analytics",
				"default":false,
				"sku":{"name":"NKR_10","display_name":"NKR-10","cpu":"2","ram":8589934592,"sort_order":2},
				"replicas_per_cell":2,
				"autoscaling":true,
				"max_replicas_per_cell":4,
				"target_cpu_utilization":70,
				"state":"provisioning",
				"created_at":"2026-08-04T12:00:00Z",
				"updated_at":"2026-08-05T12:00:00Z"
			}`))

		case r.Method == http.MethodPost && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/routers":
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body, qt.DeepEquals, map[string]interface{}{
				"name":              "analytics",
				"router_size":       "NKR_10",
				"replicas_per_cell": float64(2),
			})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"type":"neki_router",
				"name":"analytics",
				"default":false,
				"sku":{"name":"NKR_10","display_name":"NKR-10","cpu":"2","ram":8589934592,"sort_order":2},
				"replicas_per_cell":2,
				"autoscaling":false,
				"max_replicas_per_cell":null,
				"target_cpu_utilization":null,
				"state":"provisioning",
				"created_at":"2026-08-04T12:00:00Z",
				"updated_at":"2026-08-04T12:00:00Z"
			}`))

		case r.Method == http.MethodPatch && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/routers/analytics":
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body, qt.DeepEquals, map[string]interface{}{
				"autoscaling":            true,
				"max_replicas_per_cell":  float64(4),
				"target_cpu_utilization": float64(70),
				"parameters": map[string]interface{}{
					"router": map[string]interface{}{"replication-lag-tolerable-max": "15m"},
				},
			})
			_, _ = w.Write([]byte(`{
				"type":"neki_router",
				"name":"analytics",
				"default":false,
				"sku":{"name":"NKR_10","display_name":"NKR-10","cpu":"2","ram":8589934592,"sort_order":2},
				"replicas_per_cell":2,
				"autoscaling":true,
				"max_replicas_per_cell":4,
				"target_cpu_utilization":70,
				"state":"updating",
				"created_at":"2026-08-04T12:00:00Z",
				"updated_at":"2026-08-05T12:00:00Z"
			}`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/router-size-skus":
			c.Assert(r.URL.Query().Get("rates"), qt.Equals, "true")
			_, _ = w.Write([]byte(`[{
				"name":"NKR_5","display_name":"NKR-5","cpu":"1","ram":4294967296,"sort_order":1,"rate":25
			}]`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/routers/analytics/parameters":
			_, _ = w.Write([]byte(`[{
				"name":"replication-lag-tolerable-max",
				"display_name":"Replication lag tolerable max",
				"namespace":"router",
				"advanced":false,
				"category":null,
				"description":"Maximum tolerated replication lag",
				"parameter_type":"duration",
				"default_value":"10m",
				"value":"15m",
				"required":false,
				"created_at":"2026-08-04T12:00:00Z",
				"updated_at":"2026-08-05T12:00:00Z",
				"restart":false,
				"url":"https://planetscale.com/docs"
			}]`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/routers/analytics/changes":
			c.Assert(r.URL.Query().Get("period"), qt.Equals, "24h")
			c.Assert(r.URL.Query().Get("completed_at"), qt.Equals, "2026-08-04")
			c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
			c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "25")
			_, _ = w.Write([]byte(`{"data":[{"id":"change-1","state":"pending","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z","router_size":"NKR_10","previous_router_size":"NKR_5","replicas_per_cell":2,"previous_replicas_per_cell":1,"parameters":{"router":{"replication-lag-tolerable-max":"15m"}},"previous_parameters":{"router":{"replication-lag-tolerable-max":"10m"}}}]}`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/routers/analytics/changes/change-1":
			_, _ = w.Write([]byte(`{"id":"change-1","state":"pending","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z","router_size":"NKR_10","previous_router_size":"NKR_5","replicas_per_cell":2,"previous_replicas_per_cell":1}`))

		case r.Method == http.MethodDelete && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/routers/analytics/changes/change-1":
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodDelete && r.URL.Path == "/v1/organizations/acme/databases/app/branches/main/routers/analytics":
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	ctx := context.Background()

	routers, err := client.NekiRouters.List(ctx, &ListNekiRoutersRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(routers, qt.HasLen, 1)
	c.Assert(routers[0].Name, qt.Equals, "default")
	c.Assert(routers[0].Default, qt.IsTrue)
	c.Assert(routers[0].State, qt.Equals, "ready")
	c.Assert(routers[0].SKU.Name, qt.Equals, "NKR_5")

	router, err := client.NekiRouters.Get(ctx, &GetNekiRouterRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Router:       "analytics",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(router.Name, qt.Equals, "analytics")
	c.Assert(router.Autoscaling, qt.IsTrue)
	c.Assert(*router.MaxReplicasPerCell, qt.Equals, 4)
	c.Assert(*router.TargetCPUUtilization, qt.Equals, 70)

	replicas := 2
	router, err = client.NekiRouters.Create(ctx, &CreateNekiRouterRequest{
		Organization:    "acme",
		Database:        "app",
		Branch:          "main",
		Name:            "analytics",
		RouterSize:      "NKR_10",
		ReplicasPerCell: &replicas,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(router.Name, qt.Equals, "analytics")
	c.Assert(router.State, qt.Equals, "provisioning")

	autoscaling := true
	maxReplicas := 4
	targetCPU := 70
	router, err = client.NekiRouters.Update(ctx, &UpdateNekiRouterRequest{
		Organization:         "acme",
		Database:             "app",
		Branch:               "main",
		Router:               "analytics",
		Autoscaling:          &autoscaling,
		MaxReplicasPerCell:   &maxReplicas,
		TargetCPUUtilization: &targetCPU,
		Parameters: map[string]map[string]string{
			"router": {"replication-lag-tolerable-max": "15m"},
		},
	})
	c.Assert(err, qt.IsNil)
	c.Assert(router.State, qt.Equals, "updating")
	c.Assert(router.Autoscaling, qt.IsTrue)

	skus, err := client.NekiRouters.ListSizeSKUs(ctx, &ListNekiRouterSizeSKUsRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Rates:        true,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(skus, qt.HasLen, 1)
	c.Assert(skus[0].Name, qt.Equals, "NKR_5")
	c.Assert(skus[0].DisplayName, qt.Equals, "NKR-5")
	c.Assert(*skus[0].Rate, qt.Equals, int64(25))

	parameters, err := client.NekiRouters.ListParameters(ctx, &ListNekiRouterParametersRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Router:       "analytics",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(parameters, qt.HasLen, 1)
	c.Assert(parameters[0].Name, qt.Equals, "replication-lag-tolerable-max")
	c.Assert(parameters[0].Namespace, qt.Equals, "router")
	c.Assert(parameters[0].Value, qt.Equals, "15m")

	changes, err := client.NekiRouters.ListChanges(ctx, &ListNekiRouterChangesRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Router:       "analytics",
		Period:       "24h",
		CompletedAt:  "2026-08-04",
		Page:         2,
		PerPage:      25,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(changes, qt.HasLen, 1)
	c.Assert(changes[0].ID, qt.Equals, "change-1")
	c.Assert(changes[0].RouterSize, qt.Equals, "NKR_10")
	c.Assert(changes[0].PreviousRouterSize, qt.Equals, "NKR_5")

	change, err := client.NekiRouters.GetChange(ctx, &GetNekiRouterChangeRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Router:       "analytics",
		Change:       "change-1",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(change.State, qt.Equals, "pending")
	c.Assert(change.ReplicasPerCell, qt.Equals, 2)

	err = client.NekiRouters.CancelChange(ctx, &CancelNekiRouterChangeRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Router:       "analytics",
		Change:       "change-1",
	})
	c.Assert(err, qt.IsNil)

	err = client.NekiRouters.Delete(ctx, &DeleteNekiRouterRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Router:       "analytics",
	})
	c.Assert(err, qt.IsNil)
}
