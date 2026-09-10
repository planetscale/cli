package branch

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

const dataTopologyDiagramFixture = `{
  "authoritative_shard_group": "authoritative",
  "shard_indexes": {
    "xxhash_product_id": {"type": "xxhash", "columns": ["product_id"]},
    "xxhash_tenant_id": {"type": "xxhash", "columns": ["tenant_id"]}
  },
  "shard_groups": [
    {
      "uid": "authoritative",
      "key_ranges": [{"shard_uid": "shard-meta"}]
    },
    {
      "uid": "tenant_data",
      "default_shard_index": "xxhash_tenant_id",
      "key_ranges": [
        {"shard_uid": "shard-a", "start": "00", "end": "40"},
        {"shard_uid": "shard-b", "start": "40", "end": "80"},
        {"shard_uid": "shard-c", "start": "80", "end": "c0"},
        {"shard_uid": "shard-d", "start": "c0"}
      ]
    },
    {
      "uid": "catalog",
      "default_shard_index": "xxhash_product_id",
      "key_ranges": [
        {"shard_uid": "shard-a", "start": "00", "end": "40"},
        {"shard_uid": "shard-b", "start": "40", "end": "80"},
        {"shard_uid": "shard-c", "start": "80", "end": "c0"},
        {"shard_uid": "shard-d", "start": "c0"}
      ]
    }
  ],
  "databases": {
    "commerce": {
      "schemas": {
        "public": {
          "tables": {
            "tenants": {"shard_group_uid": "tenant_data"},
            "orders": {"shard_group_uid": "tenant_data"},
            "products": {"shard_group_uid": "catalog"},
            "price_history": {"shard_group_uid": "catalog"},
            "countries": {"shard_group_uid": "authoritative"}
          }
        }
      }
    }
  }
}`

func dataTopologyTestHelper(org string, format printer.Format, buf *bytes.Buffer, svc ps.DatabaseBranchesService) *cmdutil.Helper {
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(buf)
	p.SetHumanOutput(buf)

	return &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{DatabaseBranches: svc}, nil
		},
	}
}

func TestBranch_DataTopologyGetCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	topology := &ps.DataTopology{
		ID:           "topology-id",
		Type:         "NekiDataTopology",
		DataTopology: json.RawMessage(`{"authoritative_shard_group":"default"}`),
	}
	svc := &mock.DatabaseBranchesService{
		DataTopologyFn: func(ctx context.Context, req *ps.BranchDataTopologyRequest) (*ps.DataTopology, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			c.Assert(req.Database, qt.Equals, "my-db")
			c.Assert(req.Branch, qt.Equals, "main")
			return topology, nil
		},
	}
	ch := dataTopologyTestHelper("my-org", printer.JSON, &buf, svc)

	cmd := GetDataTopologyCmd(ch)
	cmd.SetArgs([]string{"my-db", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DataTopologyFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, topology)
}

func TestBranch_DataTopologyGetCmd_Human(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	topology := &ps.DataTopology{
		DataTopology: json.RawMessage(`{"authoritative_shard_group":"default"}`),
	}
	svc := &mock.DatabaseBranchesService{
		DataTopologyFn: func(ctx context.Context, req *ps.BranchDataTopologyRequest) (*ps.DataTopology, error) {
			return topology, nil
		},
	}
	ch := dataTopologyTestHelper("my-org", printer.Human, &buf, svc)

	cmd := GetDataTopologyCmd(ch)
	cmd.SetArgs([]string{"my-db", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"authoritative_shard_group": "default",
	})
}

func TestBranch_DataTopologyListCmd_Human(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	topology := &ps.DataTopology{DataTopology: json.RawMessage(dataTopologyDiagramFixture)}
	svc := &mock.DatabaseBranchesService{
		DataTopologyFn: func(ctx context.Context, req *ps.BranchDataTopologyRequest) (*ps.DataTopology, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			c.Assert(req.Database, qt.Equals, "my-db")
			c.Assert(req.Branch, qt.Equals, "main")
			return topology, nil
		},
	}
	ch := dataTopologyTestHelper("my-org", printer.Human, &buf, svc)

	cmd := ListDataTopologyCmd(ch)
	cmd.SetArgs([]string{"my-db", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.DataTopologyFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Equals, `my-org/my-db/main
├── databases
│   └── commerce
│       └── schemas
│           └── public
│               └── tables
│                   ├── countries → authoritative
│                   ├── orders → tenant_data
│                   ├── price_history → catalog
│                   ├── products → catalog
│                   └── tenants → tenant_data
├── shard groups
│   ├── authoritative [authoritative]
│   │   └── key ranges
│   │       └── [-∞, +∞) → shard-meta
│   ├── catalog → xxhash_product_id
│   │   └── key ranges
│   │       ├── [00, 40) → shard-a
│   │       ├── [40, 80) → shard-b
│   │       ├── [80, c0) → shard-c
│   │       └── [c0, +∞) → shard-d
│   └── tenant_data → xxhash_tenant_id
│       └── key ranges
│           ├── [00, 40) → shard-a
│           ├── [40, 80) → shard-b
│           ├── [80, c0) → shard-c
│           └── [c0, +∞) → shard-d
└── shard indexes
    ├── xxhash_product_id: xxhash(product_id)
    └── xxhash_tenant_id: xxhash(tenant_id)
`)
}

func TestDataTopologyTablesTree_ShowsPlacementInTableLabel(t *testing.T) {
	c := qt.New(t)

	database := dataTopologyDatabase{
		DefaultShardGroup: "tenant_data",
		GlobalSecondaryIndexes: map[string]dataTopologyGlobalIndex{
			"users_email_idx": {Schema: "public", Table: "users_email_lookup"},
		},
	}
	schema := dataTopologySchema{
		Tables: map[string]dataTopologyTable{
			"users": {
				PrimaryIndexes: []dataTopologyPrimaryIndex{
					{ShardIndex: "xxhash_tenant_id", Columns: []string{"tenant_id"}},
				},
				SecondaryIndexes: []dataTopologySecondaryIndex{
					{Name: "users_email_idx", Columns: []string{"email"}},
				},
			},
		},
	}

	c.Assert(renderDataTopologyTree(tablesTree("public", schema, database, "authoritative")), qt.Equals, `tables
└── users → tenant_data (database default)
    ├── primary index → xxhash_tenant_id (tenant_id)
    └── secondary index → users_email_idx (email) → public.users_email_lookup
`)
}

func TestBranch_DataTopologyListCmd_Human_Shards(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	topology := &ps.DataTopology{DataTopology: json.RawMessage(dataTopologyDiagramFixture)}
	svc := &mock.DatabaseBranchesService{
		DataTopologyFn: func(ctx context.Context, req *ps.BranchDataTopologyRequest) (*ps.DataTopology, error) {
			return topology, nil
		},
	}
	ch := dataTopologyTestHelper("my-org", printer.Human, &buf, svc)

	cmd := ListDataTopologyCmd(ch)
	cmd.SetArgs([]string{"my-db", "main", "--shards"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.Equals, `my-org/my-db/main
├── shard-a
│   ├── catalog [00, 40)
│   │   ├── commerce.public.price_history
│   │   └── commerce.public.products
│   └── tenant_data [00, 40)
│       ├── commerce.public.orders
│       └── commerce.public.tenants
├── shard-b
│   ├── catalog [40, 80)
│   │   ├── commerce.public.price_history
│   │   └── commerce.public.products
│   └── tenant_data [40, 80)
│       ├── commerce.public.orders
│       └── commerce.public.tenants
├── shard-c
│   ├── catalog [80, c0)
│   │   ├── commerce.public.price_history
│   │   └── commerce.public.products
│   └── tenant_data [80, c0)
│       ├── commerce.public.orders
│       └── commerce.public.tenants
├── shard-d
│   ├── catalog [c0, +∞)
│   │   ├── commerce.public.price_history
│   │   └── commerce.public.products
│   └── tenant_data [c0, +∞)
│       ├── commerce.public.orders
│       └── commerce.public.tenants
└── shard-meta
    └── authoritative [-∞, +∞) [authoritative]
        └── commerce.public.countries
`)
}

func TestBranch_DataTopologyListCmd_JSON(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	syncedAt := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	topology := &ps.DataTopology{
		DataTopology: json.RawMessage(dataTopologyDiagramFixture),
		SyncedAt:     &syncedAt,
	}
	svc := &mock.DatabaseBranchesService{
		DataTopologyFn: func(ctx context.Context, req *ps.BranchDataTopologyRequest) (*ps.DataTopology, error) {
			return topology, nil
		},
	}
	ch := dataTopologyTestHelper("my-org", printer.JSON, &buf, svc)

	cmd := ListDataTopologyCmd(ch)
	cmd.SetArgs([]string{"my-db", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	var output dataTopologyListOutput
	c.Assert(json.Unmarshal(buf.Bytes(), &output), qt.IsNil)
	c.Assert(output.Organization, qt.Equals, "my-org")
	c.Assert(output.Database, qt.Equals, "my-db")
	c.Assert(output.Branch, qt.Equals, "main")
	c.Assert(output.SyncedAt, qt.DeepEquals, &syncedAt)
	c.Assert(output.AuthoritativeShardGroup, qt.Equals, "authoritative")
	c.Assert(output.Databases, qt.HasLen, 1)
	c.Assert(output.Databases["commerce"].Schemas["public"].Tables, qt.HasLen, 5)
	c.Assert(output.Databases["commerce"].Schemas["public"].Tables["countries"].ShardGroupUID, qt.Equals, "authoritative")
	c.Assert(output.ShardGroups, qt.HasLen, 3)
	c.Assert(output.ShardGroups[1].UID, qt.Equals, "tenant_data")
	c.Assert(output.ShardGroups[1].DefaultShardIndex, qt.Equals, "xxhash_tenant_id")
	c.Assert(output.ShardGroups[1].KeyRanges, qt.HasLen, 4)
	c.Assert(output.ShardIndexes, qt.HasLen, 2)
	c.Assert(output.ShardIndexes["xxhash_product_id"].Columns, qt.DeepEquals, []string{"product_id"})
	c.Assert(buf.String(), qt.Not(qt.Contains), `"tree"`)
}

func TestBranch_DataTopologyListCmd_JSON_Shards(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	topology := &ps.DataTopology{DataTopology: json.RawMessage(dataTopologyDiagramFixture)}
	svc := &mock.DatabaseBranchesService{
		DataTopologyFn: func(ctx context.Context, req *ps.BranchDataTopologyRequest) (*ps.DataTopology, error) {
			return topology, nil
		},
	}
	ch := dataTopologyTestHelper("my-org", printer.JSON, &buf, svc)

	cmd := ListDataTopologyCmd(ch)
	cmd.SetArgs([]string{"my-db", "main", "--shards"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	var output dataTopologyShardsOutput
	c.Assert(json.Unmarshal(buf.Bytes(), &output), qt.IsNil)
	c.Assert(output.Organization, qt.Equals, "my-org")
	c.Assert(output.Database, qt.Equals, "my-db")
	c.Assert(output.Branch, qt.Equals, "main")
	c.Assert(output.AuthoritativeShardGroup, qt.Equals, "authoritative")
	c.Assert(output.Shards, qt.HasLen, 5)
	c.Assert(output.Shards["shard-a"].Placements, qt.HasLen, 2)
	start, end := "00", "40"
	c.Assert(output.Shards["shard-a"].Placements[0], qt.DeepEquals, dataTopologyPlacement{
		ShardGroup:        "catalog",
		KeyRange:          dataTopologyKeyRangeOutput{Start: &start, End: &end},
		DefaultShardIndex: "xxhash_product_id",
		Tables: []dataTopologyHostedRelation{
			{Database: "commerce", Schema: "public", Name: "price_history"},
			{Database: "commerce", Schema: "public", Name: "products"},
		},
	})
	c.Assert(output.Shards["shard-meta"].Placements, qt.DeepEquals, []dataTopologyPlacement{
		{
			ShardGroup:    "authoritative",
			Authoritative: true,
			KeyRange:      dataTopologyKeyRangeOutput{},
			Tables: []dataTopologyHostedRelation{
				{Database: "commerce", Schema: "public", Name: "countries"},
			},
		},
	})
	c.Assert(buf.String(), qt.Not(qt.Contains), `"tree"`)
}

func TestBranch_DataTopologyUpdateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	raw := `{"authoritative_shard_group":"default","shard_groups":[]}`
	syncedAt := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	topology := &ps.DataTopology{
		ID:           "topology-id",
		Type:         "NekiDataTopology",
		DataTopology: json.RawMessage(raw),
		SyncedAt:     &syncedAt,
	}
	svc := &mock.DatabaseBranchesService{
		UpdateDataTopologyFn: func(ctx context.Context, req *ps.UpdateBranchDataTopologyRequest) (*ps.DataTopology, error) {
			c.Assert(req.Organization, qt.Equals, "my-org")
			c.Assert(req.Database, qt.Equals, "my-db")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(string(req.DataTopology), qt.JSONEquals, map[string]any{
				"authoritative_shard_group": "default",
				"shard_groups":              []any{},
			})
			return topology, nil
		},
	}
	ch := dataTopologyTestHelper("my-org", printer.JSON, &buf, svc)

	cmd := UpdateDataTopologyCmd(ch)
	cmd.SetIn(strings.NewReader(raw))
	cmd.SetArgs([]string{"my-db", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateDataTopologyFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, topology)
}

func TestBranch_DataTopologyUpdateCmd_RequiresStdin(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	ch := dataTopologyTestHelper("my-org", printer.JSON, &buf, &mock.DatabaseBranchesService{})

	cmd := UpdateDataTopologyCmd(ch)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"my-db", "main"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "no data topology provided; pipe a JSON object to standard input")
}

func TestBranch_DataTopologyUpdateCmd_RequiresJSONObject(t *testing.T) {
	c := qt.New(t)

	for _, input := range []string{"not-json", `[]`, `null`} {
		c.Run(input, func(c *qt.C) {
			var buf bytes.Buffer
			ch := dataTopologyTestHelper("my-org", printer.JSON, &buf, &mock.DatabaseBranchesService{})

			cmd := UpdateDataTopologyCmd(ch)
			cmd.SetIn(strings.NewReader(input))
			cmd.SetArgs([]string{"my-db", "main"})
			err := cmd.Execute()

			c.Assert(err, qt.IsNotNil)
			c.Assert(err.Error(), qt.Contains, "invalid data topology JSON")
		})
	}
}
