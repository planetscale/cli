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
