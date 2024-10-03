package keyspace

import (
	"bytes"
	"context"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestKeyspace_ResizeCmd(t *testing.T) {
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

	krr := &ps.KeyspaceResizeRequest{
		ID:        "wantid",
		State:     "completed",
		CreatedAt: ts,
		UpdatedAt: ts,
		Actor:     nil,
	}

	svc := &mock.BranchKeyspacesService{
		ResizeFn: func(ctx context.Context, req *ps.ResizeKeyspaceRequest) (*ps.KeyspaceResizeRequest, error) {
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Keyspace, qt.Equals, keyspace)
			c.Assert(*req.ExtraReplicas, qt.Equals, uint(3))
			c.Assert(*req.ClusterSize, qt.Equals, ps.ClusterSize("PS_10"))

			return krr, nil
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

	cmd := ResizeCmd(ch)
	cmd.SetArgs([]string{db, branch, keyspace, "--additional-replicas", "3", "--cluster-size", "PS_10"})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)
	c.Assert(svc.ResizeFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, krr)
}
