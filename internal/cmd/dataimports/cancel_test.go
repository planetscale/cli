package dataimports

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"strings"
	"testing"
)

func TestImports_Cancel_FailsIfNoImport(t *testing.T) {
	c := qt.New(t)
	org := "planetscale"
	db := "employees"
	svc := &mock.DataImportsService{
		GetDataImportStatusFn: func(ctx context.Context, request *ps.GetImportStatusRequest) (*ps.DataImport, error) {
			return nil, errors.New("DataImport does not exist")
		},
	}
	expectedOut := []string{
		fmt.Sprintf("Getting current import status for PlanetScale database %s...\n", db),
	}
	shouldInvokeCancel := false
	out, err := invokeCancel(org, db, c, shouldInvokeCancel, svc)
	c.Assert(err, qt.IsNotNil)
	c.Assert(err, qt.ErrorMatches, "DataImport does not exist")
	c.Assert(out, qt.Equals, strings.Join(expectedOut, ""))
}

func TestImports_Cancel_FailsIfImportCompleted(t *testing.T) {
	c := qt.New(t)
	org := "planetscale"
	db := "employees"
	di := &ps.DataImport{
		ImportState: ps.DataImportReady,
	}
	svc := &mock.DataImportsService{
		GetDataImportStatusFn: func(ctx context.Context, request *ps.GetImportStatusRequest) (*ps.DataImport, error) {
			return di, nil
		},
	}
	expectedOut := []string{
		fmt.Sprintf("Getting current import status for PlanetScale database %s...\n", db),
	}
	shouldInvokeCancel := false
	out, err := invokeCancel(org, db, c, shouldInvokeCancel, svc)
	c.Assert(err, qt.IsNotNil)
	c.Assert(err, qt.ErrorMatches, "cannot cancel import into PlanetScale Database planetscale/employees because this import has completed")
	c.Assert(out, qt.Equals, strings.Join(expectedOut, ""))
}

func TestImports_Cancel_Success(t *testing.T) {
	c := qt.New(t)
	org := "planetscale"
	db := "employees"
	di := &ps.DataImport{
		ImportState: ps.DataImportSwitchTrafficPending,
	}
	svc := &mock.DataImportsService{
		GetDataImportStatusFn: func(ctx context.Context, request *ps.GetImportStatusRequest) (*ps.DataImport, error) {
			return di, nil
		},
		CancelDataImportFn: func(ctx context.Context, request *ps.CancelDataImportRequest) error {
			return nil
		},
	}
	expectedOut := []string{
		fmt.Sprintf("Getting current import status for PlanetScale database %s...\n", db),
		"Cancelling Data Import into PlanetScale database planetscale/employees...\n",
		"Data Import into PlanetScale database planetscale/employees has been cancelled",
	}
	shouldInvokeCancel := true
	out, err := invokeCancel(org, db, c, shouldInvokeCancel, svc)
	c.Assert(err, qt.IsNil)
	c.Assert(out, qt.Equals, strings.Join(expectedOut, ""))
}

func invokeCancel(org, dbName string, c *qt.C, shouldInvokeCancel bool, svc *mock.DataImportsService) (string, error) {
	var buf bytes.Buffer
	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&buf)
	p.SetResourceOutput(&buf)

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DataImports: svc,
			}, nil

		},
	}

	cmd := CancelDataImportCmd(ch)

	cmd.SetArgs([]string{
		"--name", dbName,
		"--force", "true",
	})
	cmd.SilenceUsage = true
	err := cmd.Execute()

	c.Assert(svc.GetDataImportStatusFnInvoked, qt.IsTrue)
	c.Assert(svc.CancelDataImportFnInvoked, qt.Equals, shouldInvokeCancel)
	return buf.String(), err
}
