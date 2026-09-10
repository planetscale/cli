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

func TestKeyspace_ConstraintsToStrategy(t *testing.T) {
	c := qt.New(t)

	// Test the helper function for translating API values to semantic strings
	c.Assert(constraintsToStrategy("maximum"), qt.Equals, "available")
	c.Assert(constraintsToStrategy("minimum"), qt.Equals, "always")
	c.Assert(constraintsToStrategy("dynamic"), qt.Equals, "lag")
	c.Assert(constraintsToStrategy("unknown"), qt.Equals, "unknown")
}

func TestKeyspace_UpdateSettingsCmd_MaxRollout(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	format := printer.JSON
	maxRollout := 8

	svc := &mock.KeyspacesService{
		UpdateSettingsFn: func(_ context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.ReplicationDurabilityConstraints, qt.IsNil)
			c.Assert(req.VReplicationFlags, qt.IsNil)
			c.Assert(req.MaxRollout, qt.Not(qt.IsNil))
			c.Assert(*req.MaxRollout, qt.Not(qt.IsNil))
			c.Assert(**req.MaxRollout, qt.Equals, maxRollout)
			return &ps.Keyspace{MaxRollout: &maxRollout}, nil
		},
	}
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"database", "main", "keyspace", "--max-rollout=8", "--reset-max-rollout=false"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"max_rollout": 8`)
}

func TestKeyspace_UpdateSettingsCmd_ResetMaxRollout(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	format := printer.JSON

	svc := &mock.KeyspacesService{
		UpdateSettingsFn: func(_ context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.ReplicationDurabilityConstraints, qt.IsNil)
			c.Assert(req.VReplicationFlags, qt.IsNil)
			c.Assert(req.MaxRollout, qt.Not(qt.IsNil))
			c.Assert(*req.MaxRollout, qt.IsNil)
			return &ps.Keyspace{}, nil
		},
	}
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"database", "main", "keyspace", "--reset-max-rollout"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsFalse)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.Contains, `"max_rollout": null`)
}

func TestKeyspace_UpdateSettingsCmd_MaxRolloutValidationBeforeAPI(t *testing.T) {
	for _, tt := range []struct {
		name      string
		args      []string
		wantError string
	}{
		{name: "too low", args: []string{"--max-rollout=0"}, wantError: `--max-rollout must be between 1 and 32`},
		{name: "too high", args: []string{"--max-rollout=33"}, wantError: `--max-rollout must be between 1 and 32`},
		{name: "set and reset", args: []string{"--max-rollout=8", "--reset-max-rollout"}, wantError: `--max-rollout and --reset-max-rollout are mutually exclusive`},
		{name: "set interactively", args: []string{"--interactive", "--max-rollout=8"}, wantError: `--max-rollout and --reset-max-rollout cannot be used with --interactive`},
		{name: "reset interactively", args: []string{"--interactive", "--reset-max-rollout"}, wantError: `--max-rollout and --reset-max-rollout cannot be used with --interactive`},
		{name: "explicit false reset interactively", args: []string{"--interactive", "--reset-max-rollout=false"}, wantError: `--max-rollout and --reset-max-rollout cannot be used with --interactive`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			format := printer.Human
			clientCalled := false
			ch := &cmdutil.Helper{
				Printer: printer.NewPrinter(&format),
				Config:  &config.Config{Organization: "planetscale"},
				Client: func() (*ps.Client, error) {
					clientCalled = true
					return nil, errors.New("unexpected API client call")
				},
			}
			cmd := UpdateSettingsCmd(ch)
			cmd.SetArgs(append([]string{"database", "main", "keyspace"}, tt.args...))
			c.Assert(cmd.Execute(), qt.ErrorMatches, tt.wantError)
			c.Assert(clientCalled, qt.IsFalse)
		})
	}
}

func TestKeyspace_UpdateSettingsCmd_ResetMaxRolloutFalseIsNoOp(t *testing.T) {
	c := qt.New(t)
	format := printer.Human
	clientCalled := false
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			clientCalled = true
			return nil, errors.New("unexpected API client call")
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"database", "main", "keyspace", "--reset-max-rollout=false"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(clientCalled, qt.IsFalse)
}

func TestKeyspace_UpdateSettingsCmd_PreservesUnspecifiedVReplicationFlags(t *testing.T) {
	c := qt.New(t)
	var buf bytes.Buffer
	format := printer.JSON
	initial := &ps.Keyspace{
		VReplicationFlags: &ps.VReplicationFlags{
			OptimizeInserts:           true,
			AllowNoBlobBinlogRowImage: true,
			VPlayerBatching:           false,
		},
	}
	updated := &ps.Keyspace{
		VReplicationFlags: &ps.VReplicationFlags{
			OptimizeInserts:           false,
			AllowNoBlobBinlogRowImage: true,
			VPlayerBatching:           false,
		},
	}
	svc := &mock.KeyspacesService{
		GetFn: func(_ context.Context, _ *ps.GetKeyspaceRequest) (*ps.Keyspace, error) {
			return initial, nil
		},
		UpdateSettingsFn: func(_ context.Context, req *ps.UpdateKeyspaceSettingsRequest) (*ps.Keyspace, error) {
			c.Assert(req.ReplicationDurabilityConstraints, qt.IsNil)
			c.Assert(req.MaxRollout, qt.IsNil)
			c.Assert(req.VReplicationFlags, qt.DeepEquals, updated.VReplicationFlags)
			return updated, nil
		},
	}
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{Keyspaces: svc}, nil
		},
	}

	cmd := UpdateSettingsCmd(ch)
	cmd.SetArgs([]string{"database", "main", "keyspace", "--vreplication-optimize-inserts=false"})
	c.Assert(cmd.Execute(), qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsTrue)
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
	cmd.SetArgs([]string{db, branch, keyspace, "--vreplication-optimize-inserts=false"})
	err := cmd.Execute()
	c.Assert(err, qt.Not(qt.IsNil)) // Just check that there is an error
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.UpdateSettingsFnInvoked, qt.IsFalse)
}
