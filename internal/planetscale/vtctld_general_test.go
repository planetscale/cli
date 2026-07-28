package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestVtctld_ListWorkflows(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/workflows")
		c.Assert(r.URL.Query().Get("keyspace"), qt.Equals, "my-keyspace")
		c.Assert(r.URL.Query().Get("workflow"), qt.Equals, "my-workflow")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":{"result":"ok"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.ListWorkflows(ctx, &VtctldListWorkflowsRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Keyspace:     "my-keyspace",
		Workflow:     "my-workflow",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"result":"ok"}`)
}

func TestVtctld_ListWorkflows_NoWorkflowFilter(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/workflows")
		c.Assert(r.URL.Query().Get("keyspace"), qt.Equals, "my-keyspace")
		c.Assert(r.URL.Query().Get("workflow"), qt.Equals, "")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":{"result":"ok"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.ListWorkflows(ctx, &VtctldListWorkflowsRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Keyspace:     "my-keyspace",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"result":"ok"}`)
}

func TestVtctld_ListWorkflows_IncludeLogs(t *testing.T) {
	tests := []struct {
		name               string
		includeLogs        *bool
		expectIncludeParam bool
		expectedInclude    string
	}{
		{name: "include_logs false", includeLogs: boolPtr(false), expectIncludeParam: true, expectedInclude: "false"},
		{name: "include_logs omitted", includeLogs: nil, expectIncludeParam: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Assert(r.Method, qt.Equals, http.MethodGet)
				c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/workflows")
				c.Assert(r.URL.Query().Get("keyspace"), qt.Equals, "my-keyspace")

				_, hasIncludeLogs := r.URL.Query()["include_logs"]
				if tt.expectIncludeParam {
					c.Assert(hasIncludeLogs, qt.IsTrue)
					c.Assert(r.URL.Query().Get("include_logs"), qt.Equals, tt.expectedInclude)
				} else {
					c.Assert(hasIncludeLogs, qt.IsFalse)
				}

				w.WriteHeader(200)
				_, err := w.Write([]byte(`{"data":{"result":"ok"}}`))
				c.Assert(err, qt.IsNil)
			}))
			defer ts.Close()

			client, err := NewClient(WithBaseURL(ts.URL))
			c.Assert(err, qt.IsNil)

			ctx := context.Background()
			data, err := client.Vtctld.ListWorkflows(ctx, &VtctldListWorkflowsRequest{
				Organization: "my-org",
				Database:     "my-db",
				Branch:       "my-branch",
				Keyspace:     "my-keyspace",
				IncludeLogs:  tt.includeLogs,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(string(data), qt.Equals, `{"result":"ok"}`)
		})
	}
}

func TestVtctld_GetRoutingRules(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/routing-rules")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":{"rules":[]}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.GetRoutingRules(ctx, &VtctldGetRoutingRulesRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"rules":[]}`)
}

func TestVtctld_GetShard(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/shard")
		c.Assert(r.URL.Query().Get("keyspace"), qt.Equals, "commerce")
		c.Assert(r.URL.Query().Get("shard"), qt.Equals, "-")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":{"keyspace":"commerce","name":"-"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.GetShard(ctx, &VtctldGetShardRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Keyspace:     "commerce",
		Shard:        "-",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"keyspace":"commerce","name":"-"}`)
}

func TestVtctld_SetShardTabletControl(t *testing.T) {
	c := qt.New(t)

	remove := true

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPut)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/shard/tablet-control")

		var body VtctldSetShardTabletControlRequest
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body.Keyspace, qt.Equals, "commerce")
		c.Assert(body.Shard, qt.Equals, "-")
		c.Assert(body.TabletType, qt.Equals, "rdonly")
		c.Assert(body.DeniedTables, qt.DeepEquals, []string{"customers"})
		c.Assert(body.Remove, qt.Not(qt.IsNil))
		c.Assert(*body.Remove, qt.Equals, true)

		w.WriteHeader(200)
		_, err = w.Write([]byte(`{"data":{}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.SetShardTabletControl(ctx, &VtctldSetShardTabletControlRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Keyspace:     "commerce",
		Shard:        "-",
		TabletType:   "rdonly",
		DeniedTables: []string{"customers"},
		Remove:       &remove,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{}`)
}

func TestVtctld_RefreshStateByShard(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/shard/refresh-state")

		var body VtctldRefreshStateByShardRequest
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body.Keyspace, qt.Equals, "commerce")
		c.Assert(body.Shard, qt.Equals, "-")
		c.Assert(body.Cells, qt.DeepEquals, []string{"zone1"})

		w.WriteHeader(200)
		_, err = w.Write([]byte(`{"data":{}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.RefreshStateByShard(ctx, &VtctldRefreshStateByShardRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Keyspace:     "commerce",
		Shard:        "-",
		Cells:        []string{"zone1"},
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{}`)
}

func TestVtctld_ListKeyspaces(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/keyspaces")
		c.Assert(r.URL.Query().Get("name"), qt.Equals, "my-keyspace")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":{"result":"ok"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.ListKeyspaces(ctx, &VtctldListKeyspacesRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Name:         "my-keyspace",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"result":"ok"}`)
}

func TestVtctld_StartWorkflow(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/workflows/my-workflow/start")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["keyspace"], qt.Equals, "my-keyspace")

		w.WriteHeader(200)
		_, err = w.Write([]byte(`{"data":{"summary":"started"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.StartWorkflow(ctx, &VtctldStartWorkflowRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Workflow:     "my-workflow",
		Keyspace:     "my-keyspace",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"summary":"started"}`)
}

func TestVtctld_StopWorkflow(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/workflows/my-workflow/stop")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["keyspace"], qt.Equals, "my-keyspace")

		w.WriteHeader(200)
		_, err = w.Write([]byte(`{"data":{"summary":"stopped"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.StopWorkflow(ctx, &VtctldStopWorkflowRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Workflow:     "my-workflow",
		Keyspace:     "my-keyspace",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"summary":"stopped"}`)
}

func TestVtctld_ListKeyspaces_NoNameFilter(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/keyspaces")
		c.Assert(r.URL.Query().Get("name"), qt.Equals, "")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":{"result":"ok"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.ListKeyspaces(ctx, &VtctldListKeyspacesRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"result":"ok"}`)
}

func float64Ptr(v float64) *float64 {
	return &v
}

func TestVtctld_GetThrottlerStatus(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/throttler/status")
		c.Assert(r.URL.Query().Get("tablet_alias"), qt.Equals, "zone1-0000000100")

		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"data":{"keyspace":"commerce","enabled":true}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.GetThrottlerStatus(ctx, &VtctldGetThrottlerStatusRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		TabletAlias:  "zone1-0000000100",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"keyspace":"commerce","enabled":true}`)
}

func TestVtctld_CheckThrottler(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/throttler/check")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["tablet_alias"], qt.Equals, "zone1-0000000100")
		c.Assert(body["app_name"], qt.Equals, "online-ddl")
		c.Assert(body["scope"], qt.Equals, "self")
		c.Assert(body["skip_request_heartbeats"], qt.Equals, true)
		c.Assert(body["ok_if_not_exists"], qt.Equals, true)

		w.WriteHeader(200)
		_, err = w.Write([]byte(`{"data":{"response_code":"THROTTLER_RESPONSE_CODE_OK"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	data, err := client.Vtctld.CheckThrottler(ctx, &VtctldCheckThrottlerRequest{
		Organization:          "my-org",
		Database:              "my-db",
		Branch:                "my-branch",
		TabletAlias:           "zone1-0000000100",
		AppName:               "online-ddl",
		Scope:                 "self",
		SkipRequestHeartbeats: boolPtr(true),
		OkIfNotExists:         boolPtr(true),
	})
	c.Assert(err, qt.IsNil)
	c.Assert(string(data), qt.Equals, `{"response_code":"THROTTLER_RESPONSE_CODE_OK"}`)
}

func TestVtctld_CheckThrottler_MinimalBody(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["tablet_alias"], qt.Equals, "zone1-0000000100")

		// Optional fields are omitted entirely when unset.
		_, hasAppName := body["app_name"]
		c.Assert(hasAppName, qt.IsFalse)
		_, hasSkip := body["skip_request_heartbeats"]
		c.Assert(hasSkip, qt.IsFalse)

		w.WriteHeader(200)
		_, err = w.Write([]byte(`{"data":{"response_code":"THROTTLER_RESPONSE_CODE_OK"}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	_, err = client.Vtctld.CheckThrottler(ctx, &VtctldCheckThrottlerRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		TabletAlias:  "zone1-0000000100",
	})
	c.Assert(err, qt.IsNil)
}

func TestVtctld_UpdateThrottlerConfig(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPut)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/branches/my-branch/vtctld/throttler/config")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["keyspace"], qt.Equals, "commerce")
		c.Assert(body["enabled"], qt.Equals, true)
		c.Assert(body["threshold"], qt.Equals, float64(2.5))

		apps, ok := body["apps"].([]interface{})
		c.Assert(ok, qt.IsTrue)
		c.Assert(len(apps), qt.Equals, 1)
		app := apps[0].(map[string]interface{})
		c.Assert(app["name"], qt.Equals, "online-ddl")
		c.Assert(app["ratio"], qt.Equals, float64(0.5))

		w.WriteHeader(200)
		_, err = w.Write([]byte(`{"data":{}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	_, err = client.Vtctld.UpdateThrottlerConfig(ctx, &VtctldUpdateThrottlerConfigRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Keyspace:     "commerce",
		Enabled:      boolPtr(true),
		Threshold:    float64Ptr(2.5),
		Apps: []VtctldThrottledAppConfig{
			{Name: "online-ddl", Ratio: float64Ptr(0.5)},
		},
	})
	c.Assert(err, qt.IsNil)
}

func TestVtctld_UpdateThrottlerConfig_DisableSendsEnabledFalse(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)

		// enabled is a plain bool, so it is always present in the body, even
		// when false. The server has no tri-state, so this must be explicit.
		_, hasEnabled := body["enabled"]
		c.Assert(hasEnabled, qt.IsTrue)
		c.Assert(body["enabled"], qt.Equals, false)

		// Optional threshold/apps are omitted when unset.
		_, hasThreshold := body["threshold"]
		c.Assert(hasThreshold, qt.IsFalse)

		w.WriteHeader(200)
		_, err = w.Write([]byte(`{"data":{}}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	_, err = client.Vtctld.UpdateThrottlerConfig(ctx, &VtctldUpdateThrottlerConfigRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
		Keyspace:     "commerce",
		Enabled:      boolPtr(false),
	})
	c.Assert(err, qt.IsNil)
}
