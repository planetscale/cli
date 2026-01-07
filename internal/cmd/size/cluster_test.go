package size

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/testutil"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestSizeCluster_ListCmd_DefaultShowsAll(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"

	mysqlSKUs := []*ps.ClusterSKU{
		{Name: "PS-10", Enabled: true, Rate: testutil.Pointer[int64](39)},
	}
	postgresSKUs := []*ps.ClusterSKU{
		{Name: "PS-10", Enabled: true, Rate: testutil.Pointer[int64](39)},
	}

	callCount := 0
	svc := &mock.OrganizationsService{
		ListClusterSKUsFn: func(ctx context.Context, req *ps.ListOrganizationClusterSKUsRequest, opts ...ps.ListOption) ([]*ps.ClusterSKU, error) {
			c.Assert(req.Organization, qt.Equals, org)
			callCount++
			// First call is MySQL (no WithPostgreSQL), second is PostgreSQL (with WithPostgreSQL)
			if callCount == 1 {
				return mysqlSKUs, nil
			}
			return postgresSKUs, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Organizations: svc,
			}, nil
		},
	}

	cmd := ListCmd(ch)
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListClusterSKUsFnInvoked, qt.IsTrue)
	c.Assert(callCount, qt.Equals, 2) // Should make 2 calls (MySQL and PostgreSQL)

	// Verify output contains clusters from both engines
	c.Assert(buf.String(), qt.Contains, "PS-10")
}

func TestSizeCluster_ListCmd_PostgreSQL(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"

	orig := []*ps.ClusterSKU{
		{Name: "PS-10", Enabled: true, Rate: testutil.Pointer[int64](39)},
	}
	svc := &mock.OrganizationsService{
		ListClusterSKUsFn: func(ctx context.Context, req *ps.ListOrganizationClusterSKUsRequest, opts ...ps.ListOption) ([]*ps.ClusterSKU, error) {
			c.Assert(req.Organization, qt.Equals, org)
			return orig, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Organizations: svc,
			}, nil
		},
	}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{"--engine", "postgresql"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListClusterSKUsFnInvoked, qt.IsTrue)

	// PostgreSQL clusters use ClusterSKUPostgres type (no engine column, has configuration and replicas)
	// Each cluster shows twice: once as HA and once as single node
	res := []*ClusterSKUPostgres{
		{orig: orig[0]},
		{orig: orig[0]},
	}

	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestSizeCluster_ListCmd_MySQL(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"

	orig := []*ps.ClusterSKU{
		{Name: "PS-10", Enabled: true, Rate: testutil.Pointer[int64](39)},
	}
	svc := &mock.OrganizationsService{
		ListClusterSKUsFn: func(ctx context.Context, req *ps.ListOrganizationClusterSKUsRequest, opts ...ps.ListOption) ([]*ps.ClusterSKU, error) {
			c.Assert(req.Organization, qt.Equals, org)
			return orig, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Organizations: svc,
			}, nil
		},
	}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{"--engine", "mysql"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListClusterSKUsFnInvoked, qt.IsTrue)

	// MySQL uses ClusterSKUMySQL type (no Configuration column)
	res := []*ClusterSKUMySQL{
		{orig: orig[0]},
	}

	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestShouldIncludeCluster(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name      string
		sku       *ps.ClusterSKU
		onlyMetal bool
		want      bool
	}{
		{
			name:      "enabled cluster with rate is included",
			sku:       &ps.ClusterSKU{Enabled: true, Rate: testutil.Pointer[int64](100)},
			onlyMetal: false,
			want:      true,
		},
		{
			name:      "disabled cluster is excluded",
			sku:       &ps.ClusterSKU{Enabled: false, Rate: testutil.Pointer[int64](100)},
			onlyMetal: false,
			want:      false,
		},
		{
			name:      "cluster without rate is excluded",
			sku:       &ps.ClusterSKU{Enabled: true, Rate: nil},
			onlyMetal: false,
			want:      false,
		},
		{
			name:      "PS-DEV cluster is excluded",
			sku:       &ps.ClusterSKU{Enabled: true, Rate: testutil.Pointer[int64](100), DisplayName: "PS-DEV"},
			onlyMetal: false,
			want:      false,
		},
		{
			name:      "metal cluster included when onlyMetal is true",
			sku:       &ps.ClusterSKU{Enabled: true, Rate: testutil.Pointer[int64](100), Metal: true},
			onlyMetal: true,
			want:      true,
		},
		{
			name:      "non-metal cluster excluded when onlyMetal is true",
			sku:       &ps.ClusterSKU{Enabled: true, Rate: testutil.Pointer[int64](100), Metal: false},
			onlyMetal: true,
			want:      false,
		},
	}

	for _, tt := range tests {
		c.Run(tt.name, func(c *qt.C) {
			got := shouldIncludeCluster(tt.sku, tt.onlyMetal)
			c.Assert(got, qt.Equals, tt.want)
		})
	}
}

func TestParseDatabaseEngine(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		input       string
		wantEngine  ps.DatabaseEngine
		wantShowAll bool
		wantErr     bool
	}{
		{input: "", wantEngine: ps.DatabaseEngineMySQL, wantShowAll: true, wantErr: false},
		{input: "mysql", wantEngine: ps.DatabaseEngineMySQL, wantShowAll: false, wantErr: false},
		{input: "postgresql", wantEngine: ps.DatabaseEnginePostgres, wantShowAll: false, wantErr: false},
		{input: "postgres", wantEngine: ps.DatabaseEnginePostgres, wantShowAll: false, wantErr: false},
		{input: "invalid", wantEngine: ps.DatabaseEngineMySQL, wantShowAll: false, wantErr: true},
	}

	for _, tt := range tests {
		c.Run(tt.input, func(c *qt.C) {
			engine, showAll, err := parseDatabaseEngine(tt.input)
			if tt.wantErr {
				c.Assert(err, qt.IsNotNil)
			} else {
				c.Assert(err, qt.IsNil)
			}
			c.Assert(engine, qt.Equals, tt.wantEngine)
			c.Assert(showAll, qt.Equals, tt.wantShowAll)
		})
	}
}

func TestPostgresSingleNodeRateCalculation(t *testing.T) {
	c := qt.New(t)

	// Test that PostgreSQL single node rate is calculated as rate/3 (rounded up)
	tests := []struct {
		name         string
		rate         int64
		expectedRate string // Expected formatted rate string
	}{
		{name: "rate divisible by 3", rate: 300, expectedRate: "$100"},
		{name: "rate not divisible by 3 rounds up", rate: 100, expectedRate: "$34"}, // 100/3 = 33.33, rounds to 34
		{name: "rate of 39", rate: 39, expectedRate: "$13"},                          // 39/3 = 13 exactly
	}

	for _, tt := range tests {
		c.Run(tt.name, func(c *qt.C) {
			sku := &ps.ClusterSKU{
				Name:    "PS-10",
				Enabled: true,
				Rate:    &tt.rate,
				Metal:   false, // Non-metal so single node is created
			}

			items := []clusterSKUWithEngine{
				{sku: sku, engine: ps.DatabaseEnginePostgres},
			}

			clusters := toGenericClusterSKUs(items, false)

			// Should have 2 entries: HA and single node
			c.Assert(len(clusters), qt.Equals, 2)

			// First is HA with full rate and replicas=2
			c.Assert(clusters[0].Configuration, qt.Equals, "highly available")
			c.Assert(clusters[0].Replicas, qt.Equals, "2")

			// Second is single node with rate/3 and replicas=0
			c.Assert(clusters[1].Configuration, qt.Equals, "single node")
			c.Assert(clusters[1].Replicas, qt.Equals, "0")
			c.Assert(clusters[1].Price, qt.Equals, tt.expectedRate)
		})
	}
}

func TestPostgresMetalOnlyShowsHA(t *testing.T) {
	c := qt.New(t)

	rate := int64(1000)
	sku := &ps.ClusterSKU{
		Name:    "PS-METAL",
		Enabled: true,
		Rate:    &rate,
		Metal:   true, // Metal cluster
	}

	items := []clusterSKUWithEngine{
		{sku: sku, engine: ps.DatabaseEnginePostgres},
	}

	clusters := toGenericClusterSKUs(items, false)

	// Metal clusters only show as HA, no single node option
	c.Assert(len(clusters), qt.Equals, 1)
	c.Assert(clusters[0].Configuration, qt.Equals, "highly available")
	c.Assert(clusters[0].Replicas, qt.Equals, "2")
}

func TestToPostgresClusterSKUs(t *testing.T) {
	c := qt.New(t)

	rate := int64(39)
	sku := &ps.ClusterSKU{
		Name:    "PS-10",
		Enabled: true,
		Rate:    &rate,
		Metal:   false,
	}

	items := []clusterSKUWithEngine{
		{sku: sku, engine: ps.DatabaseEnginePostgres},
	}

	clusters := toPostgresClusterSKUs(items, false)

	// Should have 2 entries: HA and single node
	c.Assert(len(clusters), qt.Equals, 2)

	// First is HA with replicas=2
	c.Assert(clusters[0].Configuration, qt.Equals, "highly available")
	c.Assert(clusters[0].Replicas, qt.Equals, "2")
	c.Assert(clusters[0].Price, qt.Equals, "$39")

	// Second is single node with replicas=0
	c.Assert(clusters[1].Configuration, qt.Equals, "single node")
	c.Assert(clusters[1].Replicas, qt.Equals, "0")
	c.Assert(clusters[1].Price, qt.Equals, "$13") // 39/3 = 13
}

func TestToPostgresClusterSKUs_MetalOnlyShowsHA(t *testing.T) {
	c := qt.New(t)

	rate := int64(1000)
	sku := &ps.ClusterSKU{
		Name:    "PS-METAL",
		Enabled: true,
		Rate:    &rate,
		Metal:   true, // Metal cluster
	}

	items := []clusterSKUWithEngine{
		{sku: sku, engine: ps.DatabaseEnginePostgres},
	}

	clusters := toPostgresClusterSKUs(items, false)

	// Metal clusters only show as HA, no single node option
	c.Assert(len(clusters), qt.Equals, 1)
	c.Assert(clusters[0].Configuration, qt.Equals, "highly available")
	c.Assert(clusters[0].Replicas, qt.Equals, "2")
}

func TestFormatClusterBase(t *testing.T) {
	c := qt.New(t)

	storage := int64(100 * 1024 * 1024 * 1024) // 100 GB in bytes
	rate := int64(50)
	sku := &ps.ClusterSKU{
		Name:    "PS-10",
		CPU:     "2",
		Memory:  8 * 1024 * 1024 * 1024, // 8 GB in bytes
		Storage: &storage,
		Rate:    &rate,
	}

	base := formatClusterBase(sku, ps.DatabaseEngineMySQL, nil)

	c.Assert(base.name, qt.Equals, "PS-10")
	c.Assert(base.cpu, qt.Equals, "2 vCPUs")
	c.Assert(base.memory, qt.Equals, "8 GB")
	c.Assert(base.storage, qt.Equals, "100 GB")
	c.Assert(base.price, qt.Equals, "$50")
	c.Assert(base.engine, qt.Equals, "mysql")

	// Test with rate override
	overrideRate := int64(25)
	baseOverride := formatClusterBase(sku, ps.DatabaseEnginePostgres, &overrideRate)

	c.Assert(baseOverride.price, qt.Equals, "$25")
	c.Assert(baseOverride.engine, qt.Equals, "postgresql")
}
