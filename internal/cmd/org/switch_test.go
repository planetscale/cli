package org

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/testutil"
	ps "github.com/planetscale/planetscale-go/planetscale"

	qt "github.com/frankban/quicktest"
)

func TestOrganization_SwitchCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)

	organization := "planetscale"
	testfs := testutil.MemFS{}

	svc := &mock.OrganizationsService{
		GetFn: func(ctx context.Context, req *ps.GetOrganizationRequest) (*ps.Organization, error) {
			c.Assert(req.Organization, qt.Equals, organization)
			return &ps.Organization{Name: organization}, nil
		},
	}
	ch := &cmdutil.Helper{
		Printer:  p,
		ConfigFS: config.NewConfigFS(testfs),
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Organizations: svc,
			}, nil
		},
	}

	configPath := filepath.Join(t.TempDir(), "pscale.yml")

	cmd := SwitchCmd(ch)
	cmd.SetArgs([]string{organization, "--save-config", configPath})
	err := cmd.Execute()
	c.Assert(err, qt.IsNil)

	out, err := os.ReadFile(configPath)
	c.Assert(err, qt.IsNil)
	c.Assert(string(out), qt.Equals, fmt.Sprintf("org: %s\n", organization))
	c.Assert(buf.String(), qt.Contains, "Successfully switched to organization")
}
