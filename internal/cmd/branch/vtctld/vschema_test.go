package vtctld

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestGetVSchema(t *testing.T) {
	c := qt.New(t)

	const (
		org      = "my-org"
		db       = "my-db"
		branch   = "my-branch"
		keyspace = "commerce"
	)

	svc := &mock.VtctldService{
		GetVSchemaFn: func(ctx context.Context, req *ps.VtctldGetVSchemaRequest) (json.RawMessage, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)
			return json.RawMessage(`{"multi_tenant_spec":{"tenant_id_column_name":"source_shard_id","tenant_id_column_type":"INT64"}}`), nil
		},
	}

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Vtctld: svc}, nil
		},
	}

	cmd := GetVSchemaCmd(ch)
	cmd.SetArgs([]string{db, branch, "--keyspace", keyspace})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetVSchemaFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"multi_tenant_spec": map[string]any{
			"tenant_id_column_name": "source_shard_id",
			"tenant_id_column_type": "INT64",
		},
	})
}
