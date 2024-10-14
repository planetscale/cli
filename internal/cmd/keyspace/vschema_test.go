package keyspace

import (
	"bytes"
	"context"
	"os"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestKeyspace_VSchemaShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "main"

	raw := "{\"sharded\":true,\"tables\":{}}"

	vSchema := &ps.VSchema{
		Raw: raw,
	}

	svc := &mock.KeyspacesService{
		VSchemaFn: func(ctx context.Context, req *ps.GetKeyspaceVSchemaRequest) (*ps.VSchema, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)

			return vSchema, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Keyspaces: svc,
			}, nil
		},
	}

	cmd := ShowVSchemaCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.VSchemaFnInvoked, qt.IsTrue)

	c.Assert(buf.String(), qt.JSONEquals, vSchema)
}

func TestKeyspace_UpdateVSchemaCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "main"

	raw := "{\"sharded\":true,\"tables\":{}}"

	vSchema := &ps.VSchema{
		Raw: raw,
	}

	svc := &mock.KeyspacesService{
		UpdateVSchemaFn: func(ctx context.Context, req *ps.UpdateKeyspaceVSchemaRequest) (*ps.VSchema, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)
			c.Assert(req.VSchema, qt.Equals, raw)

			return vSchema, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Keyspaces: svc,
			}, nil
		},
	}

	tmpFile, err := os.CreateTemp("", "vschema.json")
	c.Assert(err, qt.IsNil)
	tmpFile.Write([]byte(raw))
	tmpFile.Close()

	cmd := UpdateVSchemaCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace})
	cmd.Flags().Set("vschema", tmpFile.Name())

	err = cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateVSchemaFnInvoked, qt.IsTrue)

	c.Assert(buf.String(), qt.JSONEquals, vSchema)
}
