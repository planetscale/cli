package keyspace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/pflag"
)

func TestKeyspace_UpdateSettingsCmd_OnlyVReplicationFlags(t *testing.T) {
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

	rdcStrategy := "available" // from the API

	// Initial keyspace state
	ks := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: rdcStrategy,
		},
		VReplicationFlags: &ps.VReplicationFlags{
			OptimizeInserts:           true,
			AllowNoBlobBinlogRowImage: true,
			VPlayerBatching:           false,
		},
	}

	// Expected updated keyspace response
	updatedKs := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: rdcStrategy,
		},
		VReplicationFlags: &ps.VReplicationFlags{
			OptimizeInserts:           false,
			AllowNoBlobBinlogRowImage: false,
			VPlayerBatching:           true,
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
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)
			c.Assert(req.ReplicationDurabilityConstraints, qt.IsNil)
			c.Assert(req.VReplicationFlags.OptimizeInserts, qt.Equals, false)
			c.Assert(req.VReplicationFlags.AllowNoBlobBinlogRowImage, qt.Equals, false)
			c.Assert(req.VReplicationFlags.VPlayerBatching, qt.Equals, true)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{
		db,
		branch,
		keyspace,
		"--vreplication-optimize-inserts=false",
		"--vreplication-enable-noblob-binlog-mode=false",
		"--vreplication-batch-replication-events=true",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func TestKeyspace_UpdateSettingsCmd(t *testing.T) {
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

	initialRdcStrategy := "available" // from the API
	updatedRdcStrategy := "lag"       // from the API

	// Initial keyspace state
	ks := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: initialRdcStrategy,
		},
		VReplicationFlags: &ps.VReplicationFlags{
			OptimizeInserts:           true,
			AllowNoBlobBinlogRowImage: true,
			VPlayerBatching:           false,
		},
	}

	// Expected updated keyspace response - all settings changed
	updatedKs := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: updatedRdcStrategy, // changed
		},
		VReplicationFlags: &ps.VReplicationFlags{
			OptimizeInserts:           false, // changed
			AllowNoBlobBinlogRowImage: false, // changed
			VPlayerBatching:           true,  // changed
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
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)

			// All settings should be changed
			c.Assert(req.ReplicationDurabilityConstraints.Strategy, qt.Equals, updatedRdcStrategy)
			c.Assert(req.VReplicationFlags.OptimizeInserts, qt.Equals, false)
			c.Assert(req.VReplicationFlags.AllowNoBlobBinlogRowImage, qt.Equals, false)
			c.Assert(req.VReplicationFlags.VPlayerBatching, qt.Equals, true)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{
		db,
		branch,
		keyspace,
		"--replication-durability-constraints-strategy=dynamic",
		"--vreplication-optimize-inserts=false",
		"--vreplication-enable-noblob-binlog-mode=false",
		"--vreplication-batch-replication-events=true",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func TestKeyspace_UpdateSettingsCmd_OnlyDurabilityConstraints(t *testing.T) {
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

	initialRdcStrategy := "available" // from the API
	updatedRdcStrategy := "lag"       // from the API

	// We've fixed the UpdateSettingsCmd to initialize its flags properly

	// Initial keyspace state
	ks := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: initialRdcStrategy,
		},
		VReplicationFlags: &ps.VReplicationFlags{
			OptimizeInserts:           true,
			AllowNoBlobBinlogRowImage: true,
			VPlayerBatching:           false,
		},
	}

	// Expected updated keyspace response
	updatedKs := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: updatedRdcStrategy,
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
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)
			c.Assert(req.ReplicationDurabilityConstraints.Strategy, qt.Equals, updatedRdcStrategy)
			c.Assert(req.VReplicationFlags, qt.IsNil)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{
		db,
		branch,
		keyspace,
		"--replication-durability-constraints-strategy=dynamic",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func TestKeyspace_UpdateSettingsCmd_NilVReplicationFlags(t *testing.T) {
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

	rdcStrategy := "available" // from the API

	// Initial keyspace state with nil VReplicationFlags
	ks := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: rdcStrategy,
		},
		VReplicationFlags: nil, // Deliberately set to nil
	}

	// Expected updated keyspace response
	updatedKs := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: rdcStrategy,
		},
		VReplicationFlags: &ps.VReplicationFlags{
			OptimizeInserts:           false,
			AllowNoBlobBinlogRowImage: false,
			VPlayerBatching:           true,
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
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)

			// Durability constraints were not changed, so they are not sent.
			c.Assert(req.ReplicationDurabilityConstraints, qt.IsNil)

			// Check that VReplication flags are initialized (since flags were provided)
			c.Assert(req.VReplicationFlags, qt.Not(qt.IsNil))
			c.Assert(req.VReplicationFlags.OptimizeInserts, qt.Equals, false)
			c.Assert(req.VReplicationFlags.AllowNoBlobBinlogRowImage, qt.Equals, false)
			c.Assert(req.VReplicationFlags.VPlayerBatching, qt.Equals, true)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{
		db,
		branch,
		keyspace,
		"--vreplication-optimize-inserts=false",
		"--vreplication-enable-noblob-binlog-mode=false",
		"--vreplication-batch-replication-events=true",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func TestKeyspace_UpdateSettingsCmd_NilReplicationDurabilityConstraints(t *testing.T) {
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

	updatedRdcStrategy := "lag" // from the API

	// Initial keyspace state with nil ReplicationDurabilityConstraints
	ks := &ps.Keyspace{
		ID:                               "ks1",
		Name:                             keyspace,
		CreatedAt:                        ts,
		UpdatedAt:                        ts,
		ReplicationDurabilityConstraints: nil, // Deliberately set to nil
		VReplicationFlags: &ps.VReplicationFlags{
			OptimizeInserts:           true,
			AllowNoBlobBinlogRowImage: true,
			VPlayerBatching:           false,
		},
	}

	// Expected updated keyspace response
	updatedKs := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: updatedRdcStrategy,
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
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)

			// Check that durability constraints are applied despite initial nil value
			c.Assert(req.ReplicationDurabilityConstraints, qt.Not(qt.IsNil))
			c.Assert(req.ReplicationDurabilityConstraints.Strategy, qt.Equals, updatedRdcStrategy)

			// VReplication flags were not changed, so they are not sent.
			c.Assert(req.VReplicationFlags, qt.IsNil)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{
		db,
		branch,
		keyspace,
		"--replication-durability-constraints-strategy=dynamic",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func TestKeyspace_UpdateSettingsCmd_PreserveNilValues(t *testing.T) {
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

	// Initial keyspace state with both structures nil
	ks := &ps.Keyspace{
		ID:                               "ks1",
		Name:                             keyspace,
		CreatedAt:                        ts,
		UpdatedAt:                        ts,
		ReplicationDurabilityConstraints: nil, // Deliberately set to nil
		VReplicationFlags:                nil, // Deliberately set to nil
	}

	// Expected updated keyspace response - only RDC changed
	updatedKs := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: "lag", // Changed via flag
		},
		VReplicationFlags: nil, // This should remain nil
	}

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)

			return ks, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)

			// ReplicationDurabilityConstraints should be initialized (flag modified it)
			c.Assert(req.ReplicationDurabilityConstraints, qt.Not(qt.IsNil))
			c.Assert(req.ReplicationDurabilityConstraints.Strategy, qt.Equals, "lag")

			// VReplicationFlags should remain nil (no flags modified it)
			c.Assert(req.VReplicationFlags, qt.IsNil)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{
		db,
		branch,
		keyspace,
		"--replication-durability-constraints-strategy=dynamic",
	})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func boolPtr(b bool) *bool { return &b }

// Regression: the API may return no throttler at all. A threshold-only update
// must not send enabled=false and silently turn the throttler off.
func TestKeyspace_UpdateSettingsCmd_ThresholdOnlyWithNoExistingThrottler(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"

	ks := &ps.Keyspace{ID: "ks1", Name: keyspace, Throttler: nil}

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return ks, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Throttler, qt.Not(qt.IsNil))
			c.Assert(req.Throttler.Enabled, qt.IsNil)
			c.Assert(*req.Throttler.Threshold, qt.Equals, 10.0)

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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--throttler-threshold=10"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
}

func TestKeyspace_UpdateSettingsCmd_DisableThrottler(t *testing.T) {
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
	threshold := 5.0

	ks := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		Throttler: &ps.KeyspaceThrottler{Enabled: boolPtr(true), Threshold: &threshold},
	}

	updatedKs := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		Throttler: &ps.KeyspaceThrottler{Enabled: boolPtr(false), Threshold: &threshold},
	}

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return ks, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Throttler, qt.Not(qt.IsNil))
			c.Assert(req.Throttler.Enabled, qt.Not(qt.IsNil))
			c.Assert(*req.Throttler.Enabled, qt.Equals, false)
			// The threshold was not changed, so the API keeps its current value.
			c.Assert(req.Throttler.Threshold, qt.IsNil)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--throttler-enabled=false"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func TestKeyspace_UpdateSettingsCmd_ThrottlerThresholdOnly(t *testing.T) {
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
	initial := 5.0
	updated := 10.0

	ks := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		Throttler: &ps.KeyspaceThrottler{Enabled: boolPtr(true), Threshold: &initial},
	}

	updatedKs := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		Throttler: &ps.KeyspaceThrottler{Enabled: boolPtr(true), Threshold: &updated},
	}

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return ks, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			// Changing only the threshold must not disable the throttler.
			c.Assert(req.Throttler.Enabled, qt.IsNil)
			c.Assert(*req.Throttler.Threshold, qt.Equals, 10.0)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--throttler-threshold=10"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func TestKeyspace_UpdateSettingsCmd_RejectsNegativeThrottlerThreshold(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{ID: "ks1", Name: keyspace}, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--throttler-threshold=-1"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, ".*throttler-threshold must be greater than or equal to 0")
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsFalse)
}

func TestKeyspace_UpdateSettingsCmd_MaxRollout(t *testing.T) {
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
	maxRollout := 8

	updatedKs := &ps.Keyspace{
		ID:         "ks1",
		Name:       keyspace,
		CreatedAt:  ts,
		UpdatedAt:  ts,
		MaxRollout: &maxRollout,
	}

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{
				ID:   "ks1",
				Name: keyspace,
				ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
					Strategy: "",
				},
			}, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.MaxRollout, qt.Not(qt.IsNil))
			c.Assert(*req.MaxRollout, qt.Equals, 8)
			c.Assert(req.Throttler, qt.IsNil)
			c.Assert(req.ReplicationDurabilityConstraints, qt.IsNil)
			c.Assert(req.VReplicationFlags, qt.IsNil)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--max-rollout=8"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func TestKeyspace_UpdateSettingsCmd_RejectsOutOfRangeMaxRollout(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"

	for _, arg := range []string{"--max-rollout=0", "--max-rollout=33"} {
		svc := &mock.KeyspacesService{
			GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
				return &ps.Keyspace{ID: "ks1", Name: keyspace}, nil
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

		cmd := UpdateSettingsCmd(ch)
		cmd.SetArgs([]string{db, branch, keyspace, arg})
		err := cmd.Execute()
		c.Assert(err, qt.ErrorMatches, ".*max-rollout must be between 1 and 32")
		c.Assert(svc.GetFnInvoked, qt.IsFalse)
		c.Assert(svc.UpdateSettingsFnInvoked, qt.IsFalse)
	}
}

func TestKeyspace_UpdateSettingsCmd_OmittedMaxRolloutIsNotSent(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{ID: "ks1", Name: keyspace}, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.MaxRollout, qt.IsNil)
			c.Assert(req.ReplicationDurabilityConstraints, qt.IsNil)
			c.Assert(*req.Throttler.Threshold, qt.Equals, 10.0)

			return &ps.Keyspace{ID: "ks1", Name: keyspace}, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--throttler-threshold=10"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
}

func TestKeyspace_UpdateSettingsCmd_ThrottlerOnlyOmitsEmptyDurabilityStrategy(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"

	ks := &ps.Keyspace{
		ID:   "ks1",
		Name: keyspace,
		ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
			Strategy: "",
		},
	}

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return ks, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.ReplicationDurabilityConstraints, qt.IsNil)
			c.Assert(req.VReplicationFlags, qt.IsNil)
			c.Assert(req.MaxRollout, qt.IsNil)
			c.Assert(*req.Throttler.Threshold, qt.Equals, 6.0)

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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--throttler-threshold=6"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
}

func TestKeyspace_UpdateSettingsCmd_MaxRolloutAndThrottlerSkipGet(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{
				ID: "ks1",
				ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
					Strategy: "",
				},
			}, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(*req.MaxRollout, qt.Equals, 4)
			c.Assert(*req.Throttler.Threshold, qt.Equals, 6.0)
			c.Assert(req.Throttler.Enabled, qt.IsNil)
			c.Assert(req.ReplicationDurabilityConstraints, qt.IsNil)
			c.Assert(req.VReplicationFlags, qt.IsNil)
			return &ps.Keyspace{ID: "ks1"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "sharded", "--max-rollout=4", "--throttler-threshold=6"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
}

func TestKeyspace_UpdateSettingsCmd_MaxRolloutAndVReplicationFetchesOnce(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{
				ID: "ks1",
				VReplicationFlags: &ps.VReplicationFlags{
					OptimizeInserts:           true,
					AllowNoBlobBinlogRowImage: true,
					VPlayerBatching:           true,
				},
				ReplicationDurabilityConstraints: &ps.ReplicationDurabilityConstraints{
					Strategy: "",
				},
			}, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(*req.MaxRollout, qt.Equals, 4)
			c.Assert(req.VReplicationFlags.OptimizeInserts, qt.Equals, true)
			c.Assert(req.VReplicationFlags.AllowNoBlobBinlogRowImage, qt.Equals, true)
			c.Assert(req.VReplicationFlags.VPlayerBatching, qt.Equals, false)
			c.Assert(req.ReplicationDurabilityConstraints, qt.IsNil)
			c.Assert(req.Throttler, qt.IsNil)
			return &ps.Keyspace{ID: "ks1"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "sharded", "--max-rollout=4", "--vreplication-batch-replication-events=false"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
}

func TestKeyspace_UpdateSettingsCmd_Storage(t *testing.T) {
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

	updatedKs := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		Storage: &ps.KeyspaceStorage{
			StorageBytes:        107374182400,
			MaxStorageBytes:     4398046511104,
			DiskScalingStrategy: "grow",
		},
	}

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{ID: "ks1", Name: keyspace}, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Storage, qt.Not(qt.IsNil))
			c.Assert(req.Storage.DiskScalingStrategy, qt.Not(qt.IsNil))
			c.Assert(*req.Storage.DiskScalingStrategy, qt.Equals, "grow")
			c.Assert(req.Storage.MaxStorageBytes, qt.Not(qt.IsNil))
			c.Assert(*req.Storage.MaxStorageBytes, qt.Equals, int64(4398046511104))
			c.Assert(req.Storage.StorageBytes, qt.IsNil)
			c.Assert(req.ReplicationDurabilityConstraints, qt.IsNil)
			c.Assert(req.VReplicationFlags, qt.IsNil)
			c.Assert(req.Throttler, qt.IsNil)
			c.Assert(req.MaxRollout, qt.IsNil)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--disk-scaling-strategy=grow", "--max-storage=4398046511104"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func TestKeyspace_UpdateSettingsCmd_StorageIOPS(t *testing.T) {
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
	iops := int64(6000)
	throughput := int64(500)

	updatedKs := &ps.Keyspace{
		ID:        "ks1",
		Name:      keyspace,
		CreatedAt: ts,
		UpdatedAt: ts,
		Storage: &ps.KeyspaceStorage{
			StorageBytes:        107374182400,
			MaxStorageBytes:     4398046511104,
			DiskScalingStrategy: "grow",
			IOPS:                &iops,
			ThroughputMiBs:      &throughput,
		},
	}

	svc := &mock.KeyspacesService{
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Storage, qt.Not(qt.IsNil))
			c.Assert(req.Storage.IOPS, qt.Not(qt.IsNil))
			c.Assert(*req.Storage.IOPS, qt.Equals, iops)
			c.Assert(req.Storage.ThroughputMiBs, qt.Not(qt.IsNil))
			c.Assert(*req.Storage.ThroughputMiBs, qt.Equals, throughput)
			c.Assert(req.Storage.DiskScalingStrategy, qt.IsNil)
			c.Assert(req.Storage.MaxStorageBytes, qt.IsNil)
			c.Assert(req.Storage.StorageBytes, qt.IsNil)
			c.Assert(req.Throttler, qt.IsNil)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--storage-iops=6000", "--storage-throughput=500"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func TestKeyspace_UpdateSettingsCmd_StorageIOPSMustBePositive(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.KeyspacesService{}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "sharded", "--storage-iops=0"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, "--storage-iops must be greater than 0")
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsFalse)
}

func TestKeyspace_UpdateSettingsCmd_StorageIONotApplied(t *testing.T) {
	iops := int64(3000)
	throughput := int64(125)

	tests := []struct {
		name    string
		args    []string
		storage *ps.KeyspaceStorage
		wantErr string
	}{
		{
			// The API leaves the values out when configurable disk I/O is
			// not enabled for the organization.
			name:    "iops left out of the response",
			args:    []string{"--storage-iops=6000"},
			storage: &ps.KeyspaceStorage{DiskScalingStrategy: "grow"},
			wantErr: "--storage-iops was not applied: configurable disk IOPS and throughput are not enabled for this keyspace. Any other requested settings were applied",
		},
		{
			name:    "throughput left out of the response",
			args:    []string{"--storage-throughput=500"},
			storage: &ps.KeyspaceStorage{DiskScalingStrategy: "grow"},
			wantErr: "--storage-throughput was not applied: configurable disk IOPS and throughput are not enabled for this keyspace. Any other requested settings were applied",
		},
		{
			name:    "no storage in the response",
			args:    []string{"--storage-iops=6000"},
			wantErr: "--storage-iops was not applied: .*",
		},
		{
			name:    "iops differs from the request",
			args:    []string{"--storage-iops=6000"},
			storage: &ps.KeyspaceStorage{IOPS: &iops, ThroughputMiBs: &throughput},
			wantErr: "--storage-iops was not applied: the keyspace reports 3000 instead of 6000. Any other requested settings were applied",
		},
		{
			name:    "throughput differs from the request",
			args:    []string{"--storage-throughput=500"},
			storage: &ps.KeyspaceStorage{IOPS: &iops, ThroughputMiBs: &throughput},
			wantErr: "--storage-throughput was not applied: the keyspace reports 125 instead of 500. Any other requested settings were applied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			var buf bytes.Buffer
			format := printer.JSON
			p := printer.NewPrinter(&format)
			p.SetResourceOutput(&buf)

			svc := &mock.KeyspacesService{
				UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
					return &ps.Keyspace{ID: "ks1", Name: "sharded", Storage: tt.storage}, nil
				},
			}

			ch := &cmdutil.Helper{
				Printer: p,
				Config:  &config.Config{Organization: "planetscale"},
				Client: func() (*ps.Client, error) {
					return &ps.Client{Keyspaces: svc}, nil
				},
			}

			cmd := UpdateSettingsCmd(ch)
			cmd.SetArgs(append([]string{"planetscale", "main", "sharded"}, tt.args...))
			err := cmd.Execute()
			c.Assert(err, qt.ErrorMatches, tt.wantErr)
			c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
			c.Assert(buf.String(), qt.Equals, "")
		})
	}
}

func TestKeyspace_UpdateSettingsCmd_MaxStorageAcceptsHumanReadableSize(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	tests := []struct {
		arg   string
		bytes int64
	}{
		{arg: "200GiB", bytes: 214748364800},
		{arg: "1TiB", bytes: 1099511627776},
		{arg: "1.5 TiB", bytes: 1649267441664},
		{arg: "107374182400", bytes: 107374182400},
		// Plain integers are parsed exactly, including above 2^53 where a
		// float64 would round.
		{arg: "9007199254740993", bytes: 9007199254740993},
	}

	for _, tt := range tests {
		svc := &mock.KeyspacesService{
			UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
				c.Assert(req.Storage, qt.Not(qt.IsNil))
				c.Assert(req.Storage.MaxStorageBytes, qt.Not(qt.IsNil))
				c.Assert(*req.Storage.MaxStorageBytes, qt.Equals, tt.bytes)

				return &ps.Keyspace{ID: "ks1", Name: "sharded"}, nil
			},
		}

		ch := &cmdutil.Helper{
			Printer: p,
			Config:  &config.Config{Organization: "planetscale"},
			Client: func() (*ps.Client, error) {
				return &ps.Client{Keyspaces: svc}, nil
			},
		}

		cmd := UpdateSettingsCmd(ch)
		cmd.SetArgs([]string{"planetscale", "main", "sharded", "--max-storage=" + tt.arg})
		err := cmd.Execute()
		c.Assert(err, qt.IsNil)
		c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	}
}

func TestKeyspace_UpdateSettingsCmd_StorageAcceptsHumanReadableSize(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	tests := []struct {
		arg   string
		bytes int64
	}{
		{arg: "200GiB", bytes: 214748364800},
		{arg: "10TiB", bytes: 10995116277760},
		{arg: "1.5 TiB", bytes: 1649267441664},
		{arg: "214748364800", bytes: 214748364800},
	}

	for _, tt := range tests {
		svc := &mock.KeyspacesService{
			UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
				c.Assert(req.Storage, qt.Not(qt.IsNil))
				c.Assert(req.Storage.StorageBytes, qt.Not(qt.IsNil))
				c.Assert(*req.Storage.StorageBytes, qt.Equals, tt.bytes)

				return &ps.Keyspace{ID: "ks1", Name: "sharded"}, nil
			},
		}

		ch := &cmdutil.Helper{
			Printer: p,
			Config:  &config.Config{Organization: "planetscale"},
			Client: func() (*ps.Client, error) {
				return &ps.Client{Keyspaces: svc}, nil
			},
		}

		cmd := UpdateSettingsCmd(ch)
		cmd.SetArgs([]string{"planetscale", "main", "sharded", "--disk-scaling-strategy=shrink", "--storage=" + tt.arg})
		err := cmd.Execute()
		c.Assert(err, qt.IsNil)
		c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	}
}

func TestKeyspace_UpdateSettingsCmd_ShrinkStorage(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"

	updatedKs := &ps.Keyspace{
		ID:   "ks1",
		Name: keyspace,
		Storage: &ps.KeyspaceStorage{
			StorageBytes:        214748364800,
			DiskScalingStrategy: "shrink",
		},
	}

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{ID: "ks1", Name: keyspace}, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Storage, qt.Not(qt.IsNil))
			c.Assert(*req.Storage.DiskScalingStrategy, qt.Equals, "shrink")
			c.Assert(req.Storage.StorageBytes, qt.Not(qt.IsNil))
			c.Assert(*req.Storage.StorageBytes, qt.Equals, int64(214748364800))
			c.Assert(req.Storage.MaxStorageBytes, qt.IsNil)

			return updatedKs, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--disk-scaling-strategy=shrink", "--storage=214748364800"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, updatedKs)
}

func TestKeyspace_UpdateSettingsCmd_DisableDiskScaling(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "main"
	keyspace := "sharded"

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{ID: "ks1", Name: keyspace}, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Storage, qt.Not(qt.IsNil))
			c.Assert(*req.Storage.DiskScalingStrategy, qt.Equals, "disable")
			c.Assert(req.Storage.MaxStorageBytes, qt.IsNil)
			c.Assert(req.Storage.StorageBytes, qt.IsNil)

			return &ps.Keyspace{
				ID:   "ks1",
				Name: keyspace,
				Storage: &ps.KeyspaceStorage{
					DiskScalingStrategy: "disable",
				},
			}, nil
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--disk-scaling-strategy=disable"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
}

func TestKeyspace_UpdateSettingsCmd_RejectsInvalidDiskScalingStrategy(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{ID: "ks1", Name: "sharded"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "sharded", "--disk-scaling-strategy=nonsense"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, `invalid --disk-scaling-strategy "nonsense", must be one of: grow, disable, shrink`)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsFalse)
}

func TestKeyspace_UpdateSettingsCmd_StorageWithoutStrategy(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{ID: "ks1", Name: "sharded"}, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Storage, qt.Not(qt.IsNil))
			c.Assert(req.Storage.DiskScalingStrategy, qt.IsNil)
			c.Assert(req.Storage.MaxStorageBytes, qt.IsNil)
			c.Assert(req.Storage.StorageBytes, qt.Not(qt.IsNil))
			c.Assert(*req.Storage.StorageBytes, qt.Equals, int64(214748364800))

			return &ps.Keyspace{
				ID:   "ks1",
				Name: "sharded",
				Storage: &ps.KeyspaceStorage{
					StorageBytes:        214748364800,
					DiskScalingStrategy: "shrink",
				},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "sharded", "--storage=214748364800"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
}

func TestKeyspace_UpdateSettingsCmd_RejectsInvalidStorageSizes(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	tests := []struct {
		args []string
		err  string
	}{
		{
			args: []string{"--disk-scaling-strategy=shrink", "--storage=53687091201"},
			err:  "--storage must be a multiple of 1 GiB",
		},
		{
			args: []string{"--disk-scaling-strategy=shrink", "--storage=0"},
			err:  "--storage must be greater than 0",
		},
		{
			args: []string{"--disk-scaling-strategy=shrink", "--storage=lots"},
			err:  `invalid --storage "lots": must be a size in bytes or with a unit, such as 200GiB`,
		},
		{
			// Decimal units are not GiB-aligned.
			args: []string{"--disk-scaling-strategy=shrink", "--storage=10TB"},
			err:  "--storage must be a multiple of 1 GiB",
		},
		{
			args: []string{"--disk-scaling-strategy=shrink", "--storage=-1"},
			err:  "--storage must be greater than 0",
		},
		{
			args: []string{"--max-storage=0"},
			err:  "--max-storage must be greater than 0",
		},
		{
			args: []string{"--max-storage=-1073741824"},
			err:  "--max-storage must be greater than 0",
		},
		{
			args: []string{"--max-storage=9EiB"},
			err:  `invalid --max-storage "9EiB": value is too large`,
		},
		{
			args: []string{"--max-storage=lots"},
			err:  `invalid --max-storage "lots": must be a size in bytes or with a unit, such as 200GiB`,
		},
	}

	for _, tt := range tests {
		svc := &mock.KeyspacesService{
			GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
				return &ps.Keyspace{ID: "ks1", Name: "sharded"}, nil
			},
		}

		ch := &cmdutil.Helper{
			Printer: p,
			Config:  &config.Config{Organization: "planetscale"},
			Client: func() (*ps.Client, error) {
				return &ps.Client{Keyspaces: svc}, nil
			},
		}

		cmd := UpdateSettingsCmd(ch)
		cmd.SetArgs(append([]string{"planetscale", "main", "sharded"}, tt.args...))
		err := cmd.Execute()
		c.Assert(err, qt.ErrorMatches, tt.err)
		c.Assert(svc.GetFnInvoked, qt.IsFalse)
		c.Assert(svc.UpdateSettingsFnInvoked, qt.IsFalse)
	}
}

func TestKeyspace_UpdateSettingsCmd_StorageAndThrottlerSkipGet(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{ID: "ks1", Name: "sharded"}, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Storage, qt.Not(qt.IsNil))
			c.Assert(*req.Storage.DiskScalingStrategy, qt.Equals, "grow")
			c.Assert(req.Throttler, qt.Not(qt.IsNil))
			c.Assert(*req.Throttler.Enabled, qt.IsFalse)
			c.Assert(req.VReplicationFlags, qt.IsNil)

			return &ps.Keyspace{ID: "ks1", Name: "sharded"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "sharded", "--disk-scaling-strategy=grow", "--throttler-enabled=false"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
}

func TestKeyspace_UpdateSettingsCmd_OmittedStorageIsNotSent(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON

	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{ID: "ks1", Name: "sharded"}, nil
		},
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.Storage, qt.IsNil)

			return &ps.Keyspace{ID: "ks1", Name: "sharded"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "sharded", "--max-rollout=4"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
}

func TestKeyspace_UpdateSettingsCmd_NoFlagsMakesNoRequests(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	svc := &mock.KeyspacesService{
		GetFn: func(ctx context.Context, req *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{ID: "ks1", Name: "sharded"}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "sharded"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsFalse)
}

func TestKeyspace_ConstraintsToStrategy(t *testing.T) {
	c := qt.New(t)

	// Test the helper function for translating API values to semantic strings
	c.Assert(constraintsToStrategy("maximum"), qt.Equals, "available")
	c.Assert(constraintsToStrategy("minimum"), qt.Equals, "always")
	c.Assert(constraintsToStrategy("dynamic"), qt.Equals, "lag")
	c.Assert(constraintsToStrategy("unknown"), qt.Equals, "unknown")
}

func TestKeyspace_UpdateSettingsCmd_ErrorNotFound(t *testing.T) {
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
			// The ErrCode function in cmdutil will extract the ps.ErrNotFound code
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

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--vreplication-batch-replication-events=true"})
	err := cmd.Execute()
	c.Assert(err, qt.Not(qt.IsNil)) // Just check that there is an error
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsFalse)
}

func TestKeyspace_UpdateSettingsCmd_ResetStorageIO(t *testing.T) {
	iops := int64(3000)
	throughput := int64(125)

	tests := []struct {
		name     string
		args     []string
		wantBody string
	}{
		{
			name:     "reset iops",
			args:     []string{"--reset-storage-iops"},
			wantBody: `{"storage":{"iops":null}}`,
		},
		{
			name:     "reset throughput",
			args:     []string{"--reset-storage-throughput"},
			wantBody: `{"storage":{"throughput_mibs":null}}`,
		},
		{
			name:     "reset both",
			args:     []string{"--reset-storage-iops", "--reset-storage-throughput"},
			wantBody: `{"storage":{"iops":null,"throughput_mibs":null}}`,
		},
		{
			name:     "reset iops and set throughput",
			args:     []string{"--reset-storage-iops", "--storage-throughput=125"},
			wantBody: `{"storage":{"iops":null,"throughput_mibs":125}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			var buf bytes.Buffer
			format := printer.JSON
			p := printer.NewPrinter(&format)
			p.SetResourceOutput(&buf)

			updatedKs := &ps.Keyspace{
				ID:   "ks1",
				Name: "sharded",
				Storage: &ps.KeyspaceStorage{
					IOPS:           &iops,
					ThroughputMiBs: &throughput,
				},
			}

			svc := &mock.KeyspacesService{
				UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
					body, err := json.Marshal(struct {
						Storage *ps.KeyspaceStorageUpdate `json:"storage"`
					}{req.Storage})
					c.Assert(err, qt.IsNil)
					c.Assert(string(body), qt.Equals, tt.wantBody)
					return updatedKs, nil
				},
			}

			ch := &cmdutil.Helper{
				Printer: p,
				Config:  &config.Config{Organization: "planetscale"},
				Client: func() (*ps.Client, error) {
					return &ps.Client{Keyspaces: svc}, nil
				},
			}

			cmd := UpdateSettingsCmd(ch)
			cmd.SetArgs(append([]string{"planetscale", "main", "sharded"}, tt.args...))
			err := cmd.Execute()
			c.Assert(err, qt.IsNil)
			c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
			c.Assert(buf.String(), qt.JSONEquals, updatedKs)
		})
	}
}

func TestKeyspace_UpdateSettingsCmd_ResetStorageIONotApplied(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	// The API leaves the values out when configurable disk I/O is not
	// enabled for the organization.
	svc := &mock.KeyspacesService{
		UpdateSettingsFn: func(ctx context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			return &ps.Keyspace{ID: "ks1", Name: "sharded", Storage: &ps.KeyspaceStorage{DiskScalingStrategy: "grow"}}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"planetscale", "main", "sharded", "--reset-storage-iops"})
	err := cmd.Execute()
	c.Assert(err, qt.ErrorMatches, "--reset-storage-iops was not applied: configurable disk IOPS and throughput are not enabled for this keyspace. Any other requested settings were applied")
}

func TestKeyspace_UpdateSettingsCmd_SetAndResetStorageIO(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "iops",
			args:    []string{"--storage-iops=6000", "--reset-storage-iops"},
			wantErr: ".*\\[reset-storage-iops storage-iops\\] were all set",
		},
		{
			name:    "throughput",
			args:    []string{"--storage-throughput=500", "--reset-storage-throughput"},
			wantErr: ".*\\[reset-storage-throughput storage-throughput\\] were all set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)

			format := printer.JSON
			p := printer.NewPrinter(&format)

			svc := &mock.KeyspacesService{}
			ch := &cmdutil.Helper{
				Printer: p,
				Config:  &config.Config{Organization: "planetscale"},
				Client: func() (*ps.Client, error) {
					return &ps.Client{Keyspaces: svc}, nil
				},
			}

			cmd := UpdateSettingsCmd(ch)
			cmd.SetArgs(append([]string{"planetscale", "main", "sharded"}, tt.args...))
			err := cmd.Execute()
			c.Assert(err, qt.ErrorMatches, tt.wantErr)
			c.Assert(svc.UpdateSettingsFnInvoked, qt.IsFalse)
		})
	}
}

func TestKeyspace_UpdateSettingsCmd_InteractiveRejectsOtherFlags(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	p := printer.NewPrinter(&format)

	svc := &mock.KeyspacesService{}
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)

	var flags []string
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Name != "interactive" {
			flags = append(flags, f.Name)
		}
	})
	c.Assert(flags, qt.Not(qt.HasLen), 0)

	for _, name := range flags {
		c.Run(name, func(c *qt.C) {
			value := "1"
			if name == "disk-scaling-strategy" {
				value = "grow"
			}
			if name == "replication-durability-constraints-strategy" {
				value = "maximum"
			}

			cmd := UpdateSettingsCmd(ch)
			cmd.SetArgs([]string{"planetscale", "main", "sharded", "--interactive", "--" + name + "=" + value})
			err := cmd.Execute()
			c.Assert(err, qt.ErrorMatches, `.*\[interactive `+name+`\] were all set.*|.*\[`+name+` interactive\] were all set.*`)
			c.Assert(svc.GetFnInvoked, qt.IsFalse)
			c.Assert(svc.UpdateSettingsFnInvoked, qt.IsFalse)
		})
	}
}
