package dataimports

import (
	"bytes"
	"context"
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

func TestImports_LintDatabase_Success(t *testing.T) {
	c := qt.New(t)

	org := "planetscale"
	externalDataSource := ps.DataImportSource{
		Database: "aws-upstream-database",
		Port:     3306,
		HostName: "rds.amazonaws.com",
		UserName: "aws-user",
		Password: "aws-password",
	}
	res := &ps.TestDataImportSourceResponse{
		CanConnect: true,
	}

	out, err := invokeLintDatabase(&externalDataSource, org, c, res)
	c.Assert(err, qt.IsNil)
	expectedOut := []string{
		fmt.Sprintf("Testing Compatibility of database %s with user %s...\n", externalDataSource.Database, externalDataSource.UserName),
		fmt.Sprintf("database %s hosted at %s is compatible and can be imported into PlanetScale!!\n", externalDataSource.Database, externalDataSource.HostName),
	}

	c.Assert(out, qt.Equals, strings.Join(expectedOut, ""))
}

func TestImports_LintDatabase_CannotConnect(t *testing.T) {
	c := qt.New(t)

	org := "planetscale"
	externalDataSource := ps.DataImportSource{
		Database: "aws-upstream-database",
		Port:     3306,
		HostName: "rds.amazonaws.com",
		UserName: "aws-user",
		Password: "aws-password",
	}

	res := &ps.TestDataImportSourceResponse{
		CanConnect:   false,
		ConnectError: "AWS RDS is down",
	}

	out, err := invokeLintDatabase(&externalDataSource, org, c, res)
	c.Assert(err, qt.IsNotNil)
	c.Assert(err, qt.ErrorMatches, res.ConnectError)
	expectedOut := []string{
		fmt.Sprintf("Testing Compatibility of database %s with user %s...\n", externalDataSource.Database, externalDataSource.UserName),
	}

	c.Assert(out, qt.Equals, strings.Join(expectedOut, ""))
}

func TestImports_LintDatabase_SchemaIncompatible(t *testing.T) {
	c := qt.New(t)

	org := "planetscale"
	externalDataSource := ps.DataImportSource{
		Database: "aws-upstream-database",
		Port:     3306,
		HostName: "rds.amazonaws.com",
		UserName: "aws-user",
		Password: "aws-password",
	}

	res := &ps.TestDataImportSourceResponse{
		CanConnect: true,
		Errors: []*ps.DataSourceIncompatibilityError{
			&ps.DataSourceIncompatibilityError{
				LintError:        "NO_PRIMARY_KEY",
				ErrorDescription: "Table \"employees\" has no primary key",
			},
			&ps.DataSourceIncompatibilityError{
				LintError:        "NO_PRIMARY_KEY",
				ErrorDescription: "Table \"departments\" has no primary key",
			},
		},
	}

	out, err := invokeLintDatabase(&externalDataSource, org, c, res)
	c.Assert(err, qt.IsNotNil)
	expectedError := `External database compatibility check failed. Fix the following errors and then try again:

• Table "employees" has no primary key
• Table "departments" has no primary key
`
	c.Assert(err, qt.ErrorMatches, expectedError)
	expectedOut := []string{
		fmt.Sprintf("Testing Compatibility of database %s with user %s...\n", externalDataSource.Database, externalDataSource.UserName),
	}

	c.Assert(out, qt.Equals, strings.Join(expectedOut, ""))
}

func invokeLintDatabase(externalDataSource *ps.DataImportSource, org string, c *qt.C, response *ps.TestDataImportSourceResponse) (string, error) {
	svc := &mock.DataImportsService{
		TestDataImportSourceFn: func(ctx context.Context, req *ps.TestDataImportSourceRequest) (*ps.TestDataImportSourceResponse, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Connection, qt.Equals, *externalDataSource)
			return response, nil
		},
	}

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

	cmd := LintExternalDataSourceCmd(ch)

	cmd.SetArgs([]string{
		"--database", externalDataSource.Database,
		"--host", externalDataSource.HostName,
		"--username", externalDataSource.UserName,
		"--password", externalDataSource.Password,
	})
	err := cmd.Execute()

	c.Assert(svc.TestDataImportSourceFnInvoked, qt.IsTrue)
	return buf.String(), err
}
