package planetscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

const testPasswordID = "4rwwvrxk2o99" // #nosec G101 - Not a password but a password identifier.

func TestPasswords_Create(t *testing.T) {
	c := qt.New(t)
	plainText := "plain-text-password"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := fmt.Sprintf(`{
    "id": "%s",
    "role": "admin",
    "plain_text": "%s",
    "name": "planetscale-go-test-password",
    "created_at": "2021-01-14T10:19:23.000Z",
		"replica": false
}`, testPasswordID, plainText)
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	password, err := client.Passwords.Create(ctx, &DatabaseBranchPasswordRequest{
		Organization: org,
		Database:     db,
		Branch:       branch,
		Role:         "admin",
	})

	want := &DatabaseBranchPassword{
		Name:     "planetscale-go-test-password",
		PublicID: testPasswordID,

		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		Role:      "admin",
		PlainText: plainText,
		Replica:   false,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(password, qt.DeepEquals, want)
}

func TestPasswords_CreateReplica(t *testing.T) {
	c := qt.New(t)
	plainText := "plain-text-replica-password"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := fmt.Sprintf(`{
    "id": "%s",
    "role": "reader",
    "plain_text": "%s",
    "name": "planetscale-go-test-replica-password",
    "created_at": "2021-01-14T10:19:23.000Z",
		"replica": true
}`, testPasswordID, plainText)
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "my-db"
	branch := "my-branch"

	password, err := client.Passwords.Create(ctx, &DatabaseBranchPasswordRequest{
		Organization: org,
		Database:     db,
		Branch:       branch,
		Role:         "reader",
		Replica:      true,
	})

	want := &DatabaseBranchPassword{
		Name:     "planetscale-go-test-replica-password",
		PublicID: testPasswordID,

		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		Role:      "reader",
		PlainText: plainText,
		Replica:   true,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(password, qt.DeepEquals, want)
}

func TestPasswords_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{
    "data":
    [
        {
            "id": "4rwwvrxk2o99",
            "name": "planetscale-go-test-password",
            "created_at": "2021-01-14T10:19:23.000Z"
        }
    ]
}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "planetscale-go-test-db"

	passwords, err := client.Passwords.List(ctx, &ListDatabaseBranchPasswordRequest{
		Organization: org,
		Database:     db,
	})

	want := []*DatabaseBranchPassword{
		{
			Name:      "planetscale-go-test-password",
			PublicID:  testPasswordID,
			CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(passwords, qt.DeepEquals, want)
}

func TestPasswords_ListBranch(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{
    "data":
    [
        {
            "id": "4rwwvrxk2o99",
            "name": "planetscale-go-test-password",
            "database_branch": {
			  "name": "my-branch"
			},
            "created_at": "2021-01-14T10:19:23.000Z"
        }
    ]
}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "planetscale-go-test-db"
	branch := "my-branch"

	passwords, err := client.Passwords.List(ctx, &ListDatabaseBranchPasswordRequest{
		Organization: org,
		Database:     db,
		Branch:       branch,
	})

	want := []*DatabaseBranchPassword{
		{
			Name: "planetscale-go-test-password",
			Branch: DatabaseBranch{
				Name: branch,
			},
			PublicID:  testPasswordID,
			CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(passwords, qt.DeepEquals, want)
}

func TestPasswords_ListEmpty(t *testing.T) {
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

	passwords, err := client.Passwords.List(ctx, &ListDatabaseBranchPasswordRequest{
		Organization: org,
		Database:     db,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(passwords, qt.HasLen, 0)
}

func TestPasswords_ListWithPagination(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify pagination parameters are included in the request
		c.Assert(r.URL.Query().Get("page"), qt.Equals, "2")
		c.Assert(r.URL.Query().Get("per_page"), qt.Equals, "50")

		w.WriteHeader(200)
		out := `{
    "data":
    [
        {
            "id": "4rwwvrxk2o99",
            "name": "planetscale-go-test-password",
            "created_at": "2021-01-14T10:19:23.000Z"
        }
    ]
}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "planetscale-go-test-db"

	passwords, err := client.Passwords.List(ctx, &ListDatabaseBranchPasswordRequest{
		Organization: org,
		Database:     db,
	}, WithPage(2), WithPerPage(50))

	want := []*DatabaseBranchPassword{
		{
			Name:      "planetscale-go-test-password",
			PublicID:  testPasswordID,
			CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(passwords, qt.DeepEquals, want)
}

func TestPasswords_ListWithFilters(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.URL.Query().Get("q"), qt.Equals, "production")
		c.Assert(r.URL.Query().Get("status"), qt.Equals, "renewable")
		_, err := w.Write([]byte(`{"data":[]}`))
		c.Assert(err, qt.IsNil)
	}))
	defer ts.Close()

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	_, err = client.Passwords.List(context.Background(), &ListDatabaseBranchPasswordRequest{
		Organization: "my-org",
		Database:     "my-db",
		Branch:       "my-branch",
	}, WithSearch("production"), WithStatus("renewable"))
	c.Assert(err, qt.IsNil)
}

func TestPasswords_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := fmt.Sprintf(`{
    "id": "%s",
    "role": "writer",
    "name": "planetscale-go-test-password",
    "created_at": "2021-01-14T10:19:23.000Z"
}`, testPasswordID)
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "planetscale-go-test-db"
	branch := "my-branch"

	password, err := client.Passwords.Get(ctx, &GetDatabaseBranchPasswordRequest{
		Organization: org,
		Database:     db,
		Branch:       branch,
		PasswordId:   testPasswordID,
	})

	want := &DatabaseBranchPassword{
		Name:      "planetscale-go-test-password",
		PublicID:  testPasswordID,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		Role:      "writer",
	}

	c.Assert(err, qt.IsNil)
	c.Assert(password, qt.DeepEquals, want)
}

func TestPasswords_Update(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, fmt.Sprintf("/v1/organizations/my-org/databases/planetscale-go-test-db/branches/my-branch/passwords/%s", testPasswordID))

		var body map[string]any
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body, qt.DeepEquals, map[string]any{
			"name":  "renamed-password",
			"cidrs": []any{"10.0.0.0/8"},
		})

		w.WriteHeader(200)
		out := fmt.Sprintf(`{
    "id": "%s",
    "role": "writer",
    "name": "renamed-password",
    "cidrs": ["10.0.0.0/8"],
    "created_at": "2021-01-14T10:19:23.000Z"
}`, testPasswordID)
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	cidrs := []string{"10.0.0.0/8"}
	password, err := client.Passwords.Update(context.Background(), &UpdateDatabaseBranchPasswordRequest{
		Organization: "my-org",
		Database:     "planetscale-go-test-db",
		Branch:       "my-branch",
		PasswordId:   testPasswordID,
		Name:         "renamed-password",
		CIDRs:        &cidrs,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(password.Name, qt.Equals, "renamed-password")
	c.Assert(password.CIDRs, qt.DeepEquals, []string{"10.0.0.0/8"})
}

func TestPasswords_UpdateClearsCIDRs(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		c.Assert(json.NewDecoder(r.Body).Decode(&body), qt.IsNil)
		c.Assert(body, qt.DeepEquals, map[string]any{"cidrs": []any{}})

		w.WriteHeader(200)
		out := fmt.Sprintf(`{"id":"%s","name":"planetscale-go-test-password","cidrs":[]}`, testPasswordID)
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	cidrs := []string{}
	password, err := client.Passwords.Update(context.Background(), &UpdateDatabaseBranchPasswordRequest{
		Organization: "my-org",
		Database:     "planetscale-go-test-db",
		Branch:       "my-branch",
		PasswordId:   testPasswordID,
		CIDRs:        &cidrs,
	})

	c.Assert(err, qt.IsNil)
	c.Assert(password.CIDRs, qt.HasLen, 0)
}

func TestPasswords_Renew(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := fmt.Sprintf(`{
    "id": "%s",
    "role": "writer",
    "name": "planetscale-go-test-password",
    "created_at": "2021-01-14T10:19:23.000Z"
}`, testPasswordID)
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()
	org := "my-org"
	db := "planetscale-go-test-db"
	branch := "my-branch"

	password, err := client.Passwords.Renew(ctx, &RenewDatabaseBranchPasswordRequest{
		Organization: org,
		Database:     db,
		Branch:       branch,
		PasswordId:   testPasswordID,
	})

	want := &DatabaseBranchPassword{
		Name:      "planetscale-go-test-password",
		PublicID:  testPasswordID,
		CreatedAt: time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC),
		Role:      "writer",
	}

	c.Assert(err, qt.IsNil)
	c.Assert(password, qt.DeepEquals, want)
}
