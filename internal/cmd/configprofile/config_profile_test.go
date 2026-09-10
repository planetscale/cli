package configprofile

import (
	"bytes"
	"context"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func configProfileTestHelper(svc ps.NekiShardConfigurationProfilesService, out *bytes.Buffer) *cmdutil.Helper {
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(out)
	return &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiShardConfigurationProfiles: svc}, nil
		},
	}
}

func testProfile() *ps.NekiShardConfigurationProfile {
	return &ps.NekiShardConfigurationProfile{
		Name:                 "metal",
		Architecture:         "arm64",
		ClusterSize:          "M1-10",
		ClusterDisplayName:   "Metal 10",
		Default:              true,
		Metal:                true,
		Replicas:             2,
		PostgresMajorVersion: 17,
		PostgresMinorVersion: 6,
		Shards:               3,
		State:                "ready",
		CreatedAt:            time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		UpdatedAt:            time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
	}
}

func TestConfigProfileListCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{
		ListFn: func(_ context.Context, req *ps.ListNekiShardConfigurationProfilesRequest) ([]*ps.NekiShardConfigurationProfile, error) {
			c.Assert(req, qt.DeepEquals, &ps.ListNekiShardConfigurationProfilesRequest{Organization: "acme", Database: "app", Branch: "main"})
			return []*ps.NekiShardConfigurationProfile{testProfile()}, nil
		},
	}

	cmd := ListCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, []map[string]interface{}{{
		"name":                   "metal",
		"architecture":           "arm64",
		"cluster_size":           "M1-10",
		"cluster_display_name":   "Metal 10",
		"default":                true,
		"metal":                  true,
		"replicas":               2,
		"postgres_major_version": 17,
		"postgres_minor_version": 6,
		"shards":                 3,
		"state":                  "ready",
		"created_at":             "2026-08-04T12:00:00Z",
		"updated_at":             "2026-08-04T12:00:00Z",
	}})
}

func TestConfigProfileListHumanHeadersAndUTC(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	profile := testProfile()
	profile.CreatedAt = time.Date(2026, 8, 4, 12, 0, 0, 0, time.FixedZone("MDT", -6*60*60))
	svc := &mock.NekiShardConfigurationProfilesService{ListFn: func(context.Context, *ps.ListNekiShardConfigurationProfilesRequest) ([]*ps.NekiShardConfigurationProfile, error) {
		return []*ps.NekiShardConfigurationProfile{profile}, nil
	}}
	ch := &cmdutil.Helper{Printer: p, Config: &config.Config{Organization: "acme"}, Client: func() (*ps.Client, error) {
		return &ps.Client{NekiShardConfigurationProfiles: svc}, nil
	}}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "CLUSTER SIZE")
	c.Assert(out.String(), qt.Contains, "POSTGRES VERSION")
	c.Assert(out.String(), qt.Contains, "MIN STORAGE")
	c.Assert(out.String(), qt.Contains, "MAX STORAGE")
	c.Assert(out.String(), qt.Contains, "STORAGE AUTOSCALING")
	c.Assert(out.String(), qt.Contains, "CREATED AT")
	c.Assert(out.String(), qt.Contains, "2026-08-04 18:00:00")
}

func TestConfigProfileCreateCmdOnlySendsChangedOptionalFlags(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{CreateFn: func(_ context.Context, req *ps.CreateNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
		c.Assert(req.Organization, qt.Equals, "acme")
		c.Assert(req.Database, qt.Equals, "app")
		c.Assert(req.Branch, qt.Equals, "main")
		c.Assert(req.Name, qt.Equals, "metal")
		c.Assert(req.ClusterSize, qt.IsNil)
		c.Assert(req.Replicas, qt.IsNotNil)
		c.Assert(*req.Replicas, qt.Equals, 0)
		c.Assert(req.PostgresMajorVersion, qt.IsNil)
		c.Assert(req.PostgresMinorVersion, qt.IsNil)
		return testProfile(), nil
	}}

	cmd := CreateCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "--replicas", "0"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}

func TestConfigProfileCreateCmdSendsStorage(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{CreateFn: func(_ context.Context, req *ps.CreateNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
		c.Assert(req.Storage, qt.DeepEquals, &ps.NekiStorage{
			MinimumStorageBytes:   int64Ptr(10737418240),
			MaximumStorageBytes:   int64Ptr(107374182400),
			StorageAutoscaling:    boolPtr(true),
			StorageIOPS:           int64Ptr(4000),
			StorageThroughputMiBs: int64Ptr(250),
		})
		return testProfile(), nil
	}}

	cmd := CreateCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "--min-storage", "10737418240", "--max-storage", "107374182400", "--storage-autoscaling", "--storage-iops", "4000", "--storage-throughput", "250"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}

func TestConfigProfileCreateCmdNormalizesClusterSize(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{CreateFn: func(_ context.Context, req *ps.CreateNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
		c.Assert(req.ClusterSize, qt.IsNotNil)
		c.Assert(*req.ClusterSize, qt.Equals, "PS_40")
		return testProfile(), nil
	}}

	cmd := CreateCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "--cluster-size", "PS-40"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
}

func TestConfigProfileCreateCmdRejectsExtraArguments(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{}

	cmd := CreateCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "extra"})

	c.Assert(cmd.Execute(), qt.ErrorMatches, `accepts 3 arg\(s\), received 4`)
	c.Assert(svc.CreateFnInvoked, qt.IsFalse)
}

func TestConfigProfileUpdateCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{UpdateFn: func(_ context.Context, req *ps.UpdateNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
		c.Assert(req.Organization, qt.Equals, "acme")
		c.Assert(req.Database, qt.Equals, "app")
		c.Assert(req.Branch, qt.Equals, "main")
		c.Assert(req.ConfigurationProfile, qt.Equals, "metal")
		c.Assert(req.Name, qt.IsNil)
		c.Assert(req.Replicas, qt.IsNotNil)
		c.Assert(*req.Replicas, qt.Equals, 2)
		c.Assert(req.Parameters, qt.DeepEquals, map[string]map[string]string{
			"pgconf":    {"max_connections": "200", "work_mem": "64MB"},
			"pgbouncer": {"default_pool_size": "20"},
		})
		return testProfile(), nil
	}}

	cmd := UpdateCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "--replicas", "2", "--parameters", "pgconf.max_connections=200", "--parameters", "pgconf.work_mem=64MB", "--parameters", "pgbouncer.default_pool_size=20"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestConfigProfileUpdateCmdNormalizesClusterSize(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{UpdateFn: func(_ context.Context, req *ps.UpdateNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
		c.Assert(req.ClusterSize, qt.IsNotNil)
		c.Assert(*req.ClusterSize, qt.Equals, "PS_40")
		return testProfile(), nil
	}}

	cmd := UpdateCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "--cluster-size", "PS-40"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestConfigProfileUpdateCmdSendsStorage(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{UpdateFn: func(_ context.Context, req *ps.UpdateNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
		c.Assert(req.Storage, qt.DeepEquals, &ps.NekiStorage{
			MinimumStorageBytes: int64Ptr(21474836480),
			StorageAutoscaling:  boolPtr(false),
		})
		c.Assert(req.Replicas, qt.IsNil)
		return testProfile(), nil
	}}

	cmd := UpdateCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "--min-storage", "21474836480", "--storage-autoscaling=false"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
}

func TestConfigProfileDefaultCommands(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{
		GetDefaultFn: func(_ context.Context, req *ps.GetDefaultNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
			c.Assert(req, qt.DeepEquals, &ps.GetDefaultNekiShardConfigurationProfileRequest{Organization: "acme", Database: "app", Branch: "main"})
			return testProfile(), nil
		},
		SetDefaultFn: func(_ context.Context, req *ps.SetDefaultNekiShardConfigurationProfileRequest) (*ps.NekiShardConfigurationProfile, error) {
			c.Assert(req, qt.DeepEquals, &ps.SetDefaultNekiShardConfigurationProfileRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal"})
			return testProfile(), nil
		},
	}

	ch := configProfileTestHelper(svc, &out)
	cmd := DefaultCmd(ch)
	cmd.SetArgs([]string{"app", "main"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetDefaultFnInvoked, qt.IsTrue)

	out.Reset()
	cmd = SetDefaultCmd(ch)
	cmd.SetArgs([]string{"app", "main", "metal"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.SetDefaultFnInvoked, qt.IsTrue)
}

func TestConfigProfileParametersCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{ListParametersFn: func(_ context.Context, req *ps.ListNekiShardConfigurationProfileParametersRequest) ([]*ps.NekiParameter, error) {
		c.Assert(req.Extension, qt.IsNotNil)
		c.Assert(*req.Extension, qt.IsTrue)
		c.Assert(req.Internal, qt.IsNil)
		return []*ps.NekiParameter{
			{Name: "work_mem", Namespace: "pgconf", Value: "64MB"},
			{Name: "default_pool_size", Namespace: "pgbouncer", Value: "20"},
		}, nil
	}}

	cmd := ParametersCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "--namespace", "pgconf", "--extension"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, []map[string]interface{}{{
		"name": "work_mem", "display_name": "", "namespace": "pgconf", "advanced": false,
		"category": nil, "description": "", "parameter_type": "", "default_value": nil, "value": "64MB",
		"required": false, "created_at": "0001-01-01T00:00:00Z", "updated_at": nil, "restart": false, "url": "",
	}})
}

func TestConfigProfileChangesCancelCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{CancelChangeFn: func(_ context.Context, req *ps.CancelNekiShardConfigurationProfileChangeRequest) error {
		c.Assert(req, qt.DeepEquals, &ps.CancelNekiShardConfigurationProfileChangeRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal", Change: "change-1"})
		return nil
	}}

	cmd := changesCancelCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "change-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"result": "change canceled", "change_id": "change-1", "configuration_profile": "metal",
	})
}

func TestConfigProfileChangesShowHumanDisplaysDifferences(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&out)
	p.SetResourceOutput(&out)
	svc := &mock.NekiShardConfigurationProfilesService{GetChangeFn: func(_ context.Context, req *ps.GetNekiShardConfigurationProfileChangeRequest) (*ps.NekiShardConfigurationProfileChange, error) {
		c.Assert(req, qt.DeepEquals, &ps.GetNekiShardConfigurationProfileChangeRequest{
			Organization:         "acme",
			Database:             "app",
			Branch:               "main",
			ConfigurationProfile: "metal",
			Change:               "change-1",
		})
		return &ps.NekiShardConfigurationProfileChange{
			ID:                  "change-1",
			State:               "applying",
			Name:                "metal-next",
			PreviousName:        "metal",
			ClusterSize:         "M1-20",
			PreviousClusterSize: "M1-10",
			Replicas:            3,
			PreviousReplicas:    2,
			Storage: &ps.NekiStorage{
				MinimumStorageBytes: int64Ptr(21474836480),
				StorageAutoscaling:  boolPtr(false),
			},
			PreviousStorage: &ps.NekiStorage{
				MinimumStorageBytes: int64Ptr(10737418240),
				StorageAutoscaling:  boolPtr(true),
			},
			CreatedAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			Parameters: map[string]map[string]string{"pgconf": {
				"log_rotation_size": "100MB",
				"max_connections":   "200",
				"work_mem":          "64MB",
			}},
			PreviousParameters: map[string]map[string]string{"pgconf": {
				"max_connections":   "100",
				"statement_timeout": "10s",
				"work_mem":          "64MB",
			}},
		}, nil
	}}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{NekiShardConfigurationProfiles: svc}, nil
		},
	}

	cmd := changesShowCmd(ch)
	cmd.SetArgs([]string{"app", "main", "metal", "change-1"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.Contains, "Changes:")
	c.Assert(out.String(), qt.Contains, "Name: metal → metal-next")
	c.Assert(out.String(), qt.Contains, "Cluster size: M1-10 → M1-20")
	c.Assert(out.String(), qt.Contains, "Replicas: 2 → 3")
	c.Assert(out.String(), qt.Contains, "Min storage: 10737418240 → 21474836480")
	c.Assert(out.String(), qt.Contains, "Storage autoscaling: true → false")
	c.Assert(out.String(), qt.Contains, "Parameter pgconf.log_rotation_size: (default) → 100MB")
	c.Assert(out.String(), qt.Contains, "Parameter pgconf.max_connections: 100 → 200")
	c.Assert(out.String(), qt.Contains, "Parameter pgconf.statement_timeout: 10s → (default)")
}

func TestConfigProfileMaintenanceCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{RunMaintenanceFn: func(_ context.Context, req *ps.RunNekiShardConfigurationProfileMaintenanceRequest) error {
		c.Assert(req, qt.DeepEquals, &ps.RunNekiShardConfigurationProfileMaintenanceRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal"})
		return nil
	}}

	cmd := MaintenanceCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.RunMaintenanceFnInvoked, qt.IsTrue)
	c.Assert(svc.RunBulkMaintenanceFnInvoked, qt.IsFalse)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"result":                 "maintenance started",
		"configuration_profiles": []interface{}{"metal"},
	})
}

func TestConfigProfileMaintenanceCmdBulk(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{RunBulkMaintenanceFn: func(_ context.Context, req *ps.RunNekiShardConfigurationProfilesMaintenanceRequest) error {
		c.Assert(req, qt.DeepEquals, &ps.RunNekiShardConfigurationProfilesMaintenanceRequest{
			Organization:              "acme",
			Database:                  "app",
			Branch:                    "main",
			ConfigurationProfileNames: []string{"metal", "analytics"},
		})
		return nil
	}}

	cmd := MaintenanceCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "analytics"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.RunBulkMaintenanceFnInvoked, qt.IsTrue)
	c.Assert(svc.RunMaintenanceFnInvoked, qt.IsFalse)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"result":                 "maintenance started",
		"configuration_profiles": []interface{}{"metal", "analytics"},
	})
}

func TestConfigProfileExtensionsEnableCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{UpdateExtensionFn: func(_ context.Context, req *ps.UpdateNekiShardConfigurationProfileExtensionRequest) (*ps.NekiExtension, error) {
		c.Assert(req, qt.DeepEquals, &ps.UpdateNekiShardConfigurationProfileExtensionRequest{
			Organization:         "acme",
			Database:             "app",
			Branch:               "main",
			ConfigurationProfile: "metal",
			Extension:            "pg_stat_statements",
			Enabled:              true,
		})
		return &ps.NekiExtension{Name: "pg_stat_statements", Enabled: true, Loader: "shared_preload_libraries"}, nil
	}}

	cmd := ExtensionsCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"enable", "app", "main", "metal", "pg_stat_statements"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateExtensionFnInvoked, qt.IsTrue)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"name":        "pg_stat_statements",
		"description": "",
		"enabled":     true,
		"internal":    false,
		"loader":      "shared_preload_libraries",
		"url":         "",
		"parameters":  nil,
	})
}

func TestConfigProfileExtensionsDisableCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{UpdateExtensionFn: func(_ context.Context, req *ps.UpdateNekiShardConfigurationProfileExtensionRequest) (*ps.NekiExtension, error) {
		c.Assert(req.Enabled, qt.IsFalse)
		return &ps.NekiExtension{Name: req.Extension, Enabled: false}, nil
	}}

	cmd := ExtensionsCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"disable", "app", "main", "metal", "vector"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.UpdateExtensionFnInvoked, qt.IsTrue)
}

func TestConfigProfileDeleteCmd(t *testing.T) {
	c := qt.New(t)
	var out bytes.Buffer
	svc := &mock.NekiShardConfigurationProfilesService{DeleteFn: func(_ context.Context, req *ps.DeleteNekiShardConfigurationProfileRequest) error {
		c.Assert(req, qt.DeepEquals, &ps.DeleteNekiShardConfigurationProfileRequest{Organization: "acme", Database: "app", Branch: "main", ConfigurationProfile: "metal"})
		return nil
	}}

	cmd := DeleteCmd(configProfileTestHelper(svc, &out))
	cmd.SetArgs([]string{"app", "main", "metal", "--force"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(out.String(), qt.JSONEquals, map[string]interface{}{
		"result": "configuration profile deleted", "configuration_profile": "metal", "branch": "main",
	})
}

func TestConfigProfileCmdSubcommands(t *testing.T) {
	cmd := ConfigProfileCmd(&cmdutil.Helper{Config: &config.Config{}})
	for _, name := range []string{"changes", "create", "default", "delete", "extensions", "list", "maintenance", "parameters", "set-default", "show", "update"} {
		found, _, err := cmd.Find([]string{name})
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, found.Name(), qt.Equals, name)
	}

	changes, _, err := cmd.Find([]string{"changes"})
	qt.Assert(t, err, qt.IsNil)
	for _, name := range []string{"cancel", "list", "show"} {
		found, _, err := changes.Find([]string{name})
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, found.Name(), qt.Equals, name)
	}

	extensions, _, err := cmd.Find([]string{"extensions"})
	qt.Assert(t, err, qt.IsNil)
	for _, name := range []string{"disable", "enable"} {
		found, _, err := extensions.Find([]string{name})
		qt.Assert(t, err, qt.IsNil)
		qt.Assert(t, found.Name(), qt.Equals, name)
	}
}

func int64Ptr(v int64) *int64 { return &v }
func boolPtr(v bool) *bool    { return &v }
