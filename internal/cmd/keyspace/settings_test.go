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
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
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

	maxRollout := 8
	fullKs.MaxRollout = &maxRollout
	fullKs.Storage = &ps.KeyspaceStorage{
		StorageBytes:        107374182400,
		MaxStorageBytes:     4398046511104,
		DiskScalingStrategy: "grow",
	}

	settings := toKeyspaceSettings(fullKs)
	c.Assert(settings.ReplicationDurabilityConstraintStrategy, qt.Equals, "maximum") // Should be translated
	c.Assert(settings.MaxRollout, qt.Equals, "8")
	c.Assert(settings.VReplicationFlags.OptimizeInserts, qt.Equals, true)
	c.Assert(settings.VReplicationFlags.AllowNoBlobBinlogRowImage, qt.Equals, true)
	c.Assert(settings.VReplicationFlags.VPlayerBatching, qt.Equals, false)
	c.Assert(settings.Storage.DiskScalingStrategy, qt.Equals, "grow")
	c.Assert(settings.Storage.StorageBytes, qt.Equals, "100 GiB")
	c.Assert(settings.Storage.MaxStorageBytes, qt.Equals, "4.0 TiB")

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
	c.Assert(nilSettings.MaxRollout, qt.Equals, "not set")
	c.Assert(nilSettings.VReplicationFlags.OptimizeInserts, qt.Equals, false) // Default values
	c.Assert(nilSettings.VReplicationFlags.AllowNoBlobBinlogRowImage, qt.Equals, false)
	c.Assert(nilSettings.VReplicationFlags.VPlayerBatching, qt.Equals, false)
	c.Assert(nilSettings.Storage.DiskScalingStrategy, qt.Equals, "not set")
	c.Assert(nilSettings.Storage.StorageBytes, qt.Equals, "not set")
	c.Assert(nilSettings.Storage.MaxStorageBytes, qt.Equals, "not set")

	// Test with a storage object that only carries a strategy, which is what
	// the API returns once autoscaling is disabled.
	disabledKs := &ps.Keyspace{
		ID:   "ks1",
		Name: "test",
		Storage: &ps.KeyspaceStorage{
			DiskScalingStrategy: "disable",
		},
	}

	disabledSettings := toKeyspaceSettings(disabledKs)
	c.Assert(disabledSettings.Storage.DiskScalingStrategy, qt.Equals, "disable")
	c.Assert(disabledSettings.Storage.StorageBytes, qt.Equals, "not set")
	c.Assert(disabledSettings.Storage.MaxStorageBytes, qt.Equals, "not set")
}
