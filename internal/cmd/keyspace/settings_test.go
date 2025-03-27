package keyspace

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestKeyspace_SettingsCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"

	ts := time.Now()

	// Create a test keyspace with both settings defined
	ks := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: "available", // API value (displays as "maximum" to user)
		},
		VReplicationFlags: &ps.VReplicationFlags{
			OptimizeInserts:           true,
			AllowNoBlobBinlogRowImage: true,
			VPlayerBatching:           false,
		},
	}

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)

			return ks, nil
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

	cmd := SettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)

	// Check that we get JSON output (the actual content is checked in the toKeyspaceSettings test)
	c.Assert(buf.String(), qt.Not(qt.Equals), "")
}

func TestKeyspace_SettingsCmd_NilSettings(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"

	ts := time.Now()

	// Create a test keyspace with nil settings
	ks := &ps.Keyspace{
		ID:                               "ks1",
		Name:                             keyspace,
		CreatedAt:                        ts,
		UpdatedAt:                        ts,
		ReplicationDurabilityConstraints: nil, // Deliberately nil
		VReplicationFlags:                nil, // Deliberately nil
	}

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)

			return ks, nil
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

	cmd := SettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)

	// Check that we get JSON output (the actual content is checked in the toKeyspaceSettings test)
	c.Assert(buf.String(), qt.Not(qt.Equals), "")
}

func TestKeyspace_SettingsCmd_NotFound(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "nonexistent"

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			// Return a simple error for the not found case
			return nil, errors.New("not found")
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

	cmd := SettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace})
	err := cmd.Execute()
	c.Assert(err, qt.Not(qt.IsNil))
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
}

func TestBuildKeyspaceSettings(t *testing.T) {
	c := qt.New(t)

	ts := time.Now()

	// Test with all settings populated
	fullKs := &ps.Keyspace{
		ID:        "ks1",
		Name:      "test",
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: "available", // API value (displays as "maximum" to user)
		},
		VReplicationFlags: &ps.VReplicationFlags{
			OptimizeInserts:           true,
			AllowNoBlobBinlogRowImage: true,
			VPlayerBatching:           false,
		},
	}

	settings := toKeyspaceSettings(fullKs)
	c.Assert(settings.ReplicationDurabilityConstraintStrategy, qt.Equals, "maximum") // Should be translated
	c.Assert(settings.VReplicationFlags.OptimizeInserts, qt.Equals, true)
	c.Assert(settings.VReplicationFlags.AllowNoBlobBinlogRowImage, qt.Equals, true)
	c.Assert(settings.VReplicationFlags.VPlayerBatching, qt.Equals, false)

	// Test with nil settings
	nilKs := &ps.Keyspace{
		ID:                               "ks1",
		Name:                             "test",
		CreatedAt:                        ts,
		UpdatedAt:                        ts,
		ReplicationDurabilityConstraints: nil,
		VReplicationFlags:                nil,
	}

	nilSettings := toKeyspaceSettings(nilKs)
	c.Assert(nilSettings.ReplicationDurabilityConstraintStrategy, qt.Equals, "not set")
	c.Assert(nilSettings.VReplicationFlags.OptimizeInserts, qt.Equals, false) // Default values
	c.Assert(nilSettings.VReplicationFlags.AllowNoBlobBinlogRowImage, qt.Equals, false)
	c.Assert(nilSettings.VReplicationFlags.VPlayerBatching, qt.Equals, false)
}
