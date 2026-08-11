package cmdutil

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestRequirePostgresDatabase_MySQL(t *testing.T) {
	c := qt.New(t)
	client := &ps.Client{
		Databases: &mock.DatabaseService{
			GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
				return &ps.Database{Name: req.Database, Kind: ps.DatabaseEngineMySQL}, nil
			},
		},
	}
	err := RequirePostgresDatabase(context.Background(), client, "org", "mydb", "PgBouncers")
	c.Assert(err, qt.ErrorMatches, `(?s).*only available for PostgreSQL.*mysql.*`)
}

func TestRequirePostgresDatabase_Postgres(t *testing.T) {
	c := qt.New(t)
	client := &ps.Client{
		Databases: &mock.DatabaseService{
			GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
				return &ps.Database{Name: req.Database, Kind: ps.DatabaseEnginePostgres}, nil
			},
		},
	}
	err := RequirePostgresDatabase(context.Background(), client, "org", "mydb", "PgBouncers")
	c.Assert(err, qt.IsNil)
}

func TestRequirePostgresDatabase_Missing(t *testing.T) {
	c := qt.New(t)
	client := &ps.Client{
		Databases: &mock.DatabaseService{
			GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
				return nil, &ps.Error{Code: ps.ErrNotFound}
			},
		},
	}
	err := RequirePostgresDatabase(context.Background(), client, "org", "missing", "PgBouncers")
	c.Assert(err, qt.ErrorMatches, `(?s).*database.*does not exist.*`)
}
