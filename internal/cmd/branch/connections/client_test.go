package connections

import (
	"context"
	"errors"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestNotFoundExplainerLooksUpBranchOnce(t *testing.T) {
	c := qt.New(t)
	lookups := 0
	branches := &mock.DatabaseBranchesService{
		GetFn: func(context.Context, *ps.GetDatabaseBranchRequest) (*ps.DatabaseBranch, error) {
			lookups++
			return &ps.DatabaseBranch{Name: "main"}, nil
		},
	}
	ch := &cmdutil.Helper{
		Config: &config.Config{Organization: "acme"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				Databases: &mock.DatabaseService{
					GetFn: func(_ context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
						return &ps.Database{Name: req.Database, Kind: ps.DatabaseEngineMySQL}, nil
					},
				},
				DatabaseBranches: branches,
			}, nil
		},
	}
	explainer := newNotFoundExplainer(ch, "shop", "main")
	notFound := &live.NotFoundError{Organization: "acme", Database: "shop", Branch: "main"}

	first := explainer.explain(context.Background(), notFound)
	second := explainer.explain(context.Background(), notFound)

	c.Assert(first, qt.ErrorMatches, `live connections are not enabled for branch main in database shop \(organization: acme\)`)
	c.Assert(second, qt.Equals, first)
	c.Assert(lookups, qt.Equals, 1)
}

func TestNotFoundExplainerPassesOtherErrorsThrough(t *testing.T) {
	c := qt.New(t)
	other := errors.New("list connections: rate limited, please retry in a moment")

	got := newNotFoundExplainer(&cmdutil.Helper{}, "shop", "main").explain(context.Background(), other)

	c.Assert(got, qt.Equals, other)
}
