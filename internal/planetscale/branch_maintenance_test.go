package planetscale

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestBranchMaintenance_Run(t *testing.T) {
	c := qt.New(t)

	wantURL := "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/planetscale-go-test-db-branch/maintenance"

	var gotBody map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.String(), qt.DeepEquals, wantURL)

		body, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(json.Unmarshal(body, &gotBody), qt.IsNil)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	err = client.BranchMaintenance.Run(ctx, &RunBranchMaintenanceRequest{
		Organization:               testOrg,
		Database:                   testDatabase,
		Branch:                     testBranch,
		UpdatePostgresMinorVersion: true,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(gotBody, qt.DeepEquals, map[string]interface{}{"update_postgres_minor_version": true})
}

func TestBranchMaintenance_Run_OmitsMinorVersionByDefault(t *testing.T) {
	c := qt.New(t)

	var gotBody map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(json.Unmarshal(body, &gotBody), qt.IsNil)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.BranchMaintenance.Run(context.Background(), &RunBranchMaintenanceRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       testBranch,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(gotBody, qt.HasLen, 0)
}

func TestBranchMaintenance_Run_Unprocessable(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		out := `{"code":"unprocessable","message":"Maintenance cannot run while a resize is in progress."}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.BranchMaintenance.Run(context.Background(), &RunBranchMaintenanceRequest{
		Organization: testOrg,
		Database:     testDatabase,
		Branch:       testBranch,
	})
	c.Assert(err, qt.ErrorMatches, ".*Maintenance cannot run while a resize is in progress.*")
}
