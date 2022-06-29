package auditlog

import (
	"bytes"
	"context"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	qt "github.com/frankban/quicktest"
)

func TestAuditLog_List(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	branch := "development"

	auditLogs := []*ps.AuditLog{
		{ActorDisplayName: "foo"},
		{ActorDisplayName: "bar"},
	}

	resp := &ps.CursorPaginatedResponse[*ps.AuditLog]{
		Data: auditLogs,
	}

	svc := &mock.AuditLogService{
		ListFn: func(ctx context.Context, req *ps.ListAuditLogsRequest, opts ...ps.ListOption) (*ps.CursorPaginatedResponse[*ps.AuditLog], error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(len(opts), qt.Equals, 2)

			return resp, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				AuditLogs: svc,
			}, nil

		},
	}

	cmd := ListCmd(ch)
	cmd.SetArgs([]string{db, branch})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)

	backups := []*AuditLog{
		{
			Actor: "foo",
			orig:  resp.Data[0],
		},
		{
			Actor: "bar",
			orig:  resp.Data[1],
		},
	}

	c.Assert(buf.String(), qt.JSONEquals, backups)
}
