package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestNekiShardConfigurationProfilesService(t *testing.T) {
	c := qt.New(t)
	const basePath = "/v1/organizations/acme/databases/app/branches/main"
	const profilesPath = basePath + "/configuration-profiles"
	const profilePath = profilesPath + "/metal"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeProfile := func() {
			_, _ = w.Write([]byte(`{"name":"metal","architecture":"arm64","cluster_size":"M1-10","cluster_display_name":"Metal 10","default":true,"metal":true,"replicas":2,"postgres_major_version":17,"postgres_minor_version":6,"shards":3,"state":"ready","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z"}`))
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == profilesPath:
			_, _ = w.Write([]byte(`[{"name":"metal","architecture":"arm64","cluster_size":"M1-10","cluster_display_name":"Metal 10","default":true,"metal":true,"replicas":2,"postgres_major_version":17,"postgres_minor_version":6,"shards":3,"state":"ready","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z"}]`))

		case r.Method == http.MethodGet && r.URL.Path == profilePath:
			writeProfile()

		case r.Method == http.MethodPost && r.URL.Path == profilesPath+"/maintenance":
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body, qt.DeepEquals, map[string]interface{}{
				"configuration_profile_names": []interface{}{"metal", "analytics"},
			})
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodPost && r.URL.Path == profilePath+"/maintenance":
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodPost && r.URL.Path == profilesPath:
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			if _, ok := body["storage"]; ok {
				c.Assert(body, qt.DeepEquals, map[string]interface{}{
					"name": "analytics",
					"storage": map[string]interface{}{
						"minimum_storage_bytes":   float64(10737418240),
						"maximum_storage_bytes":   float64(107374182400),
						"storage_autoscaling":     true,
						"storage_iops":            float64(4000),
						"storage_throughput_mibs": float64(250),
					},
				})
			} else {
				c.Assert(body, qt.DeepEquals, map[string]interface{}{
					"name":                   "metal",
					"cluster_size":           "M1-10",
					"replicas":               float64(2),
					"postgres_major_version": "17",
					"postgres_minor_version": "6",
				})
			}
			w.WriteHeader(http.StatusCreated)
			writeProfile()

		case r.Method == http.MethodPatch && r.URL.Path == profilePath+"/extensions/pg_stat_statements":
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body, qt.DeepEquals, map[string]interface{}{"enabled": true})
			_, _ = w.Write([]byte(`{"name":"pg_stat_statements","description":"Stats","enabled":true,"internal":true,"loader":"shared_preload_libraries","url":"https://example.test/extension","parameters":[]}`))

		case r.Method == http.MethodPatch && r.URL.Path == profilePath:
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			if _, ok := body["storage"]; ok {
				c.Assert(body, qt.DeepEquals, map[string]interface{}{
					"storage": map[string]interface{}{
						"minimum_storage_bytes": float64(21474836480),
						"storage_autoscaling":   false,
					},
				})
			} else {
				c.Assert(body, qt.DeepEquals, map[string]interface{}{
					"name":       "metal-default",
					"parameters": map[string]interface{}{"pgconf": map[string]interface{}{"max_connections": "200"}},
				})
			}
			writeProfile()

		case r.Method == http.MethodDelete && r.URL.Path == profilePath:
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodGet && r.URL.Path == basePath+"/default-configuration-profile":
			http.Redirect(w, r, profilePath, http.StatusFound)

		case r.Method == http.MethodPut && r.URL.Path == basePath+"/default-configuration-profile":
			var body map[string]interface{}
			c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
			c.Assert(body, qt.DeepEquals, map[string]interface{}{"configuration_profile": "metal"})
			writeProfile()

		case r.Method == http.MethodGet && r.URL.Path == profilePath+"/parameters":
			c.Assert(r.URL.Query().Get("extension"), qt.Equals, "false")
			c.Assert(r.URL.Query().Get("internal"), qt.Equals, "true")
			_, _ = w.Write([]byte(`[{"name":"shared_buffers","display_name":"Shared buffers","namespace":"pgconf","advanced":false,"category":null,"description":"Shared memory buffers","parameter_type":"bytes","default_value":"128MB","value":"256MB","required":false,"created_at":"2026-08-04T12:00:00Z","updated_at":null,"restart":true,"max":"8589934584kB","min":"128kB","units":["kB","MB","GB"],"url":"https://example.test/parameter"}]`))

		case r.Method == http.MethodGet && r.URL.Path == profilePath+"/extensions":
			_, _ = w.Write([]byte(`[{"name":"pg_stat_statements","description":"Stats","enabled":true,"internal":true,"loader":"shared_preload_libraries","url":"https://example.test/extension","parameters":[]}]`))

		case r.Method == http.MethodGet && r.URL.Path == profilePath+"/changes":
			c.Assert(r.URL.Query().Get("period"), qt.Equals, "24h")
			c.Assert(r.URL.Query().Get("completed_at"), qt.Equals, "2026-08-04")
			c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
			c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "25")
			_, _ = w.Write([]byte(`{"data":[{"id":"change-1","state":"pending","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z","name":"metal","previous_name":"default","cluster_size":"M1-10","cluster_display_name":"Metal 10","metal":true,"cluster_rank":1,"replicas":2,"parameters":{},"previous_cluster_size":"PS-10","previous_cluster_display_name":"PS 10","previous_metal":false,"previous_cluster_rank":2,"previous_replicas":0,"previous_parameters":{}}]}`))

		case r.Method == http.MethodGet && r.URL.Path == profilePath+"/changes/change-1":
			_, _ = w.Write([]byte(`{"id":"change-1","state":"pending","created_at":"2026-08-04T12:00:00Z","updated_at":"2026-08-04T12:00:00Z","name":"metal","cluster_size":"M1-10","replicas":2,"parameters":{},"previous_parameters":{}}`))

		case r.Method == http.MethodDelete && r.URL.Path == profilePath+"/changes/change-1":
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)
	ctx := context.Background()

	profiles, err := client.NekiShardConfigurationProfiles.List(ctx, &ListNekiShardConfigurationProfilesRequest{Organization: "acme", Database: "app", Branch: "main"})
	c.Assert(err, qt.IsNil)
	c.Assert(profiles, qt.HasLen, 1)
	c.Assert(profiles[0].Name, qt.Equals, "metal")

	profile, err := client.NekiShardConfigurationProfiles.Get(ctx, &GetNekiShardConfigurationProfileRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal"})
	c.Assert(err, qt.IsNil)
	c.Assert(profile.PostgresMajorVersion, qt.Equals, 17)

	clusterSize, replicas, major, minor := "M1-10", 2, "17", "6"
	_, err = client.NekiShardConfigurationProfiles.Create(ctx, &CreateNekiShardConfigurationProfileRequest{Organization: "acme", Database: "app", Branch: "main", Name: "metal", ClusterSize: &clusterSize, Replicas: &replicas, PostgresMajorVersion: &major, PostgresMinorVersion: &minor})
	c.Assert(err, qt.IsNil)

	name := "metal-default"
	_, err = client.NekiShardConfigurationProfiles.Create(ctx, &CreateNekiShardConfigurationProfileRequest{
		Organization: "acme",
		Database:     "app",
		Branch:       "main",
		Name:         "analytics",
		Storage: &NekiStorage{
			MinimumStorageBytes:   int64Ptr(10737418240),
			MaximumStorageBytes:   int64Ptr(107374182400),
			StorageAutoscaling:    boolPtr(true),
			StorageIOPS:           int64Ptr(4000),
			StorageThroughputMiBs: int64Ptr(250),
		},
	})
	c.Assert(err, qt.IsNil)

	_, err = client.NekiShardConfigurationProfiles.Update(ctx, &UpdateNekiShardConfigurationProfileRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal", Name: &name, Parameters: map[string]map[string]string{"pgconf": {"max_connections": "200"}}})
	c.Assert(err, qt.IsNil)

	_, err = client.NekiShardConfigurationProfiles.Update(ctx, &UpdateNekiShardConfigurationProfileRequest{
		Organization:         "acme",
		Database:             "app",
		Branch:               "main",
		ConfigurationProfile: "metal",
		Storage: &NekiStorage{
			MinimumStorageBytes: int64Ptr(21474836480),
			StorageAutoscaling:  boolPtr(false),
		},
	})
	c.Assert(err, qt.IsNil)

	defaultProfile, err := client.NekiShardConfigurationProfiles.GetDefault(ctx, &GetDefaultNekiShardConfigurationProfileRequest{Organization: "acme", Database: "app", Branch: "main"})
	c.Assert(err, qt.IsNil)
	c.Assert(defaultProfile.Default, qt.IsTrue)

	_, err = client.NekiShardConfigurationProfiles.SetDefault(ctx, &SetDefaultNekiShardConfigurationProfileRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal"})
	c.Assert(err, qt.IsNil)

	extensionFilter, internal := false, true
	parameters, err := client.NekiShardConfigurationProfiles.ListParameters(ctx, &ListNekiShardConfigurationProfileParametersRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal", Extension: &extensionFilter, Internal: &internal})
	c.Assert(err, qt.IsNil)
	c.Assert(parameters[0].Restart, qt.IsTrue)
	c.Assert(parameters[0].Max, qt.Equals, "8589934584kB")
	c.Assert(parameters[0].Min, qt.Equals, "128kB")
	c.Assert(parameters[0].Units, qt.DeepEquals, []string{"kB", "MB", "GB"})

	extensions, err := client.NekiShardConfigurationProfiles.ListExtensions(ctx, &ListNekiShardConfigurationProfileExtensionsRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal"})
	c.Assert(err, qt.IsNil)
	c.Assert(extensions[0].Name, qt.Equals, "pg_stat_statements")

	extension, err := client.NekiShardConfigurationProfiles.UpdateExtension(ctx, &UpdateNekiShardConfigurationProfileExtensionRequest{
		Organization:         "acme",
		Database:             "app",
		Branch:               "main",
		ConfigurationProfile: "metal",
		Extension:            "pg_stat_statements",
		Enabled:              true,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(extension.Enabled, qt.IsTrue)

	err = client.NekiShardConfigurationProfiles.RunMaintenance(ctx, &RunNekiShardConfigurationProfileMaintenanceRequest{
		Organization:         "acme",
		Database:             "app",
		Branch:               "main",
		ConfigurationProfile: "metal",
	})
	c.Assert(err, qt.IsNil)

	err = client.NekiShardConfigurationProfiles.RunBulkMaintenance(ctx, &RunNekiShardConfigurationProfilesMaintenanceRequest{
		Organization:              "acme",
		Database:                  "app",
		Branch:                    "main",
		ConfigurationProfileNames: []string{"metal", "analytics"},
	})
	c.Assert(err, qt.IsNil)

	changes, err := client.NekiShardConfigurationProfiles.ListChanges(ctx, &ListNekiShardConfigurationProfileChangesRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal", Period: "24h", CompletedAt: "2026-08-04", Page: 2, PerPage: 25})
	c.Assert(err, qt.IsNil)
	c.Assert(changes, qt.HasLen, 1)
	c.Assert(changes[0].ID, qt.Equals, "change-1")

	change, err := client.NekiShardConfigurationProfiles.GetChange(ctx, &GetNekiShardConfigurationProfileChangeRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal", Change: "change-1"})
	c.Assert(err, qt.IsNil)
	c.Assert(change.State, qt.Equals, "pending")

	err = client.NekiShardConfigurationProfiles.CancelChange(ctx, &CancelNekiShardConfigurationProfileChangeRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal", Change: "change-1"})
	c.Assert(err, qt.IsNil)

	err = client.NekiShardConfigurationProfiles.Delete(ctx, &DeleteNekiShardConfigurationProfileRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal"})
	c.Assert(err, qt.IsNil)
}

func int64Ptr(v int64) *int64 { return &v }
