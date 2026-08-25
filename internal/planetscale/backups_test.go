package planetscale

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

const testBackup = "planetscale-go-test-backup"

func TestBackups_Create(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"planetscale-go-test-backup","type":"backup","name":"planetscale-go-test-backup","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	backup, err := client.Backups.Create(ctx, &CreateBackupRequest{
		Organization: org,
		Database:     db,
		Branch:       branch,
	})

	want := &Backup{
		PublicID:  "planetscale-go-test-backup",
		Name:      testBackup,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(backup, qt.DeepEquals, want)
}

func TestBackups_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Query().Get("to"), qt.Equals, "2023-01-01T00:00:59.999Z")
		c.Assert(r.URL.Query().Get("state"), qt.Equals, "success")
		w.WriteHeader(200)
		out := `{"data":[{"id":"planetscale-go-test-backup","type":"backup","name":"planetscale-go-test-backup","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "planetscale-go-test-db"
	branch := "my-branch"

	backups, err := client.Backups.List(ctx, &ListBackupsRequest{
		Organization: org,
		Database:     db,
		Branch:       branch,
		To:           "2023-01-01T00:00:59.999Z",
		State:        "success",
	})

	want := []*Backup{{
		PublicID:  "planetscale-go-test-backup",
		Name:      testBackup,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}}

	c.Assert(err, qt.IsNil)
	c.Assert(backups, qt.DeepEquals, want)
}

func TestBackups_ListEmpty(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data":[]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "planetscale-go-test-db"
	branch := "my-branch"

	backups, err := client.Backups.List(ctx, &ListBackupsRequest{
		Organization: org,
		Database:     db,
		Branch:       branch,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(backups, qt.HasLen, 0)
}

func TestBackups_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id":"planetscale-go-test-backup","type":"backup","name":"planetscale-go-test-backup","created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "planetscale-go-test-db"
	branch := "my-branch"

	backup, err := client.Backups.Get(ctx, &GetBackupRequest{
		Organization: org,
		Database:     db,
		Branch:       branch,
		Backup:       testBackup,
	})

	want := &Backup{
		PublicID:  "planetscale-go-test-backup",
		Name:      testBackup,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(backup, qt.DeepEquals, want)
}

func TestBackups_Update(t *testing.T) {
	c := qt.New(t)

	wantURL := "/v1/organizations/my-org/databases/planetscale-go-test-db/branches/my-branch/backups/planetscale-go-test-backup"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.String(), qt.DeepEquals, wantURL)

		var body map[string]any
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body, qt.DeepEquals, map[string]any{
			"protected": true,
		})

		w.WriteHeader(200)
		out := `{"id":"planetscale-go-test-backup","type":"backup","name":"planetscale-go-test-backup","protected":true,"created_at":"2021-01-14T10:19:23.000Z","updated_at":"2021-01-14T10:19:23.000Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "planetscale-go-test-db"
	branch := "my-branch"

	backup, err := client.Backups.Update(ctx, &UpdateBackupRequest{
		Organization: org,
		Database:     db,
		Branch:       branch,
		Backup:       testBackup,
		Protected:    true,
	})

	want := &Backup{
		PublicID:  "planetscale-go-test-backup",
		Name:      testBackup,
		Protected: true,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
	}

	c.Assert(err, qt.IsNil)
	c.Assert(backup, qt.DeepEquals, want)
}
