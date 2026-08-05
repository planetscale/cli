package planetscale

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestReadOnlyRegions_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/read-only-regions")
		w.WriteHeader(200)
		out := `{
  "data": [
    {
      "id": "ror123",
      "display_name": "Europe West",
      "ready": true,
      "ready_at": "2024-01-14T10:19:23.000Z",
      "created_at": "2024-01-14T10:19:23.000Z",
      "updated_at": "2024-01-14T10:19:23.000Z",
      "region": {
        "slug": "eu-west",
        "display_name": "EU West",
        "location": "Ireland",
        "provider": "AWS",
        "enabled": true,
        "current_default": false
      }
    }
  ]
}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	regions, err := client.ReadOnlyRegions.List(context.Background(), &ListReadOnlyRegionsRequest{
		Organization: "my-org",
		Database:     "my-db",
	})
	c.Assert(err, qt.IsNil)

	want := []*ReadOnlyRegion{
		{
			ID:          "ror123",
			DisplayName: "Europe West",
			Ready:       true,
			ReadyAt:     time.Date(2024, time.January, 14, 10, 19, 23, 0, time.UTC),
			CreatedAt:   time.Date(2024, time.January, 14, 10, 19, 23, 0, time.UTC),
			UpdatedAt:   time.Date(2024, time.January, 14, 10, 19, 23, 0, time.UTC),
			Region: Region{
				Slug:      "eu-west",
				Name:      "EU West",
				Location:  "Ireland",
				Provider:  "AWS",
				Enabled:   true,
				IsDefault: false,
			},
		},
	}
	c.Assert(regions, qt.DeepEquals, want)
}

func TestPasswords_CreateReadOnlyRegion(t *testing.T) {
	c := qt.New(t)
	plainText := "plain-text-password"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		body, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)

		var got map[string]interface{}
		c.Assert(json.Unmarshal(body, &got), qt.IsNil)
		c.Assert(got["read_only_region_id"], qt.Equals, "ror123")
		c.Assert(got["role"], qt.Equals, "reader")

		w.WriteHeader(200)
		out := `{
    "id": "4rwwvrxk2o99",
    "role": "reader",
    "plain_text": "` + plainText + `",
    "name": "ror-password",
    "access_host_url": "eu-west.connect.psdb.cloud",
    "created_at": "2021-01-14T10:19:23.000Z",
    "replica": false
}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	password, err := client.Passwords.Create(context.Background(), &DatabaseBranchPasswordRequest{
		Organization:     "my-org",
		Database:         "my-db",
		Branch:           "main",
		Role:             "reader",
		Name:             "ror-password",
		ReadOnlyRegionID: "ror123",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(password.Hostname, qt.Equals, "eu-west.connect.psdb.cloud")
	c.Assert(password.Role, qt.Equals, "reader")
	c.Assert(password.PlainText, qt.Equals, plainText)
}

func TestFindReadOnlyRegion(t *testing.T) {
	c := qt.New(t)

	regions := []*ReadOnlyRegion{
		{
			ID:          "ror123",
			DisplayName: "Europe West",
			Ready:       true,
			Region:      Region{Slug: "eu-west"},
		},
		{
			ID:          "ror456",
			DisplayName: "US East",
			Ready:       true,
			Region:      Region{Slug: "us-east"},
		},
	}

	got, err := FindReadOnlyRegion(regions, "eu-west")
	c.Assert(err, qt.IsNil)
	c.Assert(got.ID, qt.Equals, "ror123")

	got, err = FindReadOnlyRegion(regions, "ror456")
	c.Assert(err, qt.IsNil)
	c.Assert(got.Region.Slug, qt.Equals, "us-east")

	got, err = FindReadOnlyRegion(regions, "Europe West")
	c.Assert(err, qt.IsNil)
	c.Assert(got.ID, qt.Equals, "ror123")

	_, err = FindReadOnlyRegion(regions, "missing")
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "not found")

	_, err = FindReadOnlyRegion(regions, "")
	c.Assert(err, qt.IsNotNil)
}
