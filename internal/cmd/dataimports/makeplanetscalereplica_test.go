package dataimports

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestImports_MakeReplica_FailsIfNoImport(t *testing.T) {
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
	shouldInvokeMakeReplica := false
	out, err := invokeMakeReplica(org, db, c, shouldInvokeMakeReplica, svc)
	c.Assert(err, qt.IsNotNil)
	c.Assert(err, qt.ErrorMatches, "DataImport does not exist")
	c.Assert(out, qt.Equals, strings.Join(expectedOut, ""))
}

func TestImports_MakeReplica_FailsIfNotPrimary(t *testing.T) {
	c := qt.New(t)
	org := "planetscale"
	db := "employees"
	di := &ps.DataImport{
		ImportState: ps.DataImportCopyingData,
	}
	svc := &mock.DataImportsService{
		GetDataImportStatusFn: func(ctx context.Context, request *ps.GetImportStatusRequest) (*ps.DataImport, error) {
			return di, nil
		},
	}
	expectedOut := []string{
		fmt.Sprintf("Getting current import status for PlanetScale database %s...\n", db),
	}
	shouldInvokeMakeReplica := false
	out, err := invokeMakeReplica(org, db, c, shouldInvokeMakeReplica, svc)
	c.Assert(err, qt.IsNotNil)
	c.Assert(err, qt.ErrorMatches, fmt.Sprintf("cannot make PlanetScale Database %s/%s Replica because it is not serving as a Primary", org, db))
	c.Assert(out, qt.Equals, strings.Join(expectedOut, ""))
}

func TestImports_MakeReplica_FailsIfAlreadyReplica(t *testing.T) {
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
	}
	expectedOut := []string{
		fmt.Sprintf("Getting current import status for PlanetScale database %s...\n", db),
	}
	shouldInvokeMakeReplica := false
	out, err := invokeMakeReplica(org, db, c, shouldInvokeMakeReplica, svc)
	c.Assert(err, qt.IsNotNil)
	c.Assert(err, qt.ErrorMatches, fmt.Sprintf("cannot make PlanetScale Database %s/%s Replica because it is not serving as a Primary", org, db))
	c.Assert(out, qt.Equals, strings.Join(expectedOut, ""))
}

func TestImports_MakeReplica_FailsIfComplete(t *testing.T) {
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
	shouldInvokeMakeReplica := false
	out, err := invokeMakeReplica(org, db, c, shouldInvokeMakeReplica, svc)
	c.Assert(err, qt.IsNotNil)
	c.Assert(err, qt.ErrorMatches, fmt.Sprintf("cannot make PlanetScale Database %s/%s Replica because this import has completed", org, db))
	c.Assert(out, qt.Equals, strings.Join(expectedOut, ""))
}

func TestImports_MakeReplica_Success(t *testing.T) {
	c := qt.New(t)
	org := "planetscale"
	db := "employees"
	di := &ps.DataImport{
		ImportState: ps.DataImportSwitchTrafficCompleted,
	}
	svc := &mock.DataImportsService{
		GetDataImportStatusFn: func(ctx context.Context, request *ps.GetImportStatusRequest) (*ps.DataImport, error) {
			return di, nil
		},
		MakePlanetScaleReplicaFn: func(ctx context.Context, request *ps.MakePlanetScaleReplicaRequest) (*ps.DataImport, error) {
			di.ImportState = ps.DataImportSwitchTrafficPending
			return di, nil
		},
	}
	expectedOut := []string{
		fmt.Sprintf("Getting current import status for PlanetScale database %s...\n", db),
		"Switching PlanetScale database employees to Primary...\n",
		"Successfully switch PlanetScale database employees to Replica.\n",
		"1. Started Data Copy\n",
		"2. Copied Data\n",
		"> 3. Running as Replica\n",
		"4. Running as Primary\n",
		"5. Detached external database\n",
	}
	shouldInvokeMakeReplica := true
	out, err := invokeMakeReplica(org, db, c, shouldInvokeMakeReplica, svc)
	c.Assert(err, qt.IsNil)
	c.Assert(out, qt.Equals, strings.Join(expectedOut, ""))
}

func invokeMakeReplica(org, dbName string, c *qt.C, shouldInvokeMakeReplica bool, svc *mock.DataImportsService) (string, error) {
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

	cmd := MakePlanetScaleReplicaCmd(ch)

	cmd.SetArgs([]string{
		"--name", dbName,
		"--force", "true",
	})
	cmd.SilenceUsage = true
	err := cmd.Execute()

	c.Assert(svc.GetDataImportStatusFnInvoked, qt.IsTrue)
	c.Assert(svc.MakePlanetScaleReplicaFnInvoked, qt.Equals, shouldInvokeMakeReplica)
	return buf.String(), err
}
