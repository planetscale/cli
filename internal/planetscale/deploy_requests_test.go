package planetscale

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestDeployRequests_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": null, "number": 1337}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	dr, err := client.DeployRequests.Get(ctx, &GetDeployRequestRequest{
		Organization: "test-organization",
		Database:     "test-database",
		Number:       1337,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := &DeployRequest{
		ID:         "test-deploy-request-id",
		Number:     1337,
		Branch:     "development",
		IntoBranch: "some-branch",
		Notes:      "",
		CreatedAt:  testTime,
		UpdatedAt:  testTime,
		ClosedAt:   nil,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(dr, qt.DeepEquals, want)
}

func TestDeployRequests_Deploy(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": "2021-01-14T10:19:23.000Z", "deployment": { "state": "queued"}, "number": 1337}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	dr, err := client.DeployRequests.Deploy(ctx, &PerformDeployRequest{
		Organization: "test-organization",
		Database:     "test-database",
		Number:       1337,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := &DeployRequest{
		ID:         "test-deploy-request-id",
		Branch:     "development",
		IntoBranch: "some-branch",
		Number:     1337,
		Deployment: &Deployment{
			State: "queued",
		},
		Notes:     "",
		CreatedAt: testTime,
		UpdatedAt: testTime,
		ClosedAt:  &testTime,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(dr, qt.DeepEquals, want)
}

func TestDeployRequests_InstantDeploy(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			InstantDDL bool `json:"instant_ddl"`
		}
		err := json.NewDecoder(r.Body).Decode(&request)
		c.Assert(err, qt.IsNil)
		c.Assert(request.InstantDDL, qt.Equals, true)
		w.WriteHeader(200)
		out := `{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": "2021-01-14T10:19:23.000Z", "deployment": { "state": "queued", "instant_ddl": true }, "number": 1337}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	dr, err := client.DeployRequests.Deploy(ctx, &PerformDeployRequest{
		Organization: "test-organization",
		Database:     "test-database",
		Number:       1337,
		InstantDDL:   true,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := &DeployRequest{
		ID:         "test-deploy-request-id",
		Branch:     "development",
		IntoBranch: "some-branch",
		Number:     1337,
		Deployment: &Deployment{
			State:      "queued",
			InstantDDL: true,
		},
		Notes:     "",
		CreatedAt: testTime,
		UpdatedAt: testTime,
		ClosedAt:  &testTime,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(dr, qt.DeepEquals, want)
}

func TestDeployRequests_DeployWithStrategy(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			InstantDDL bool   `json:"instant_ddl"`
			Strategy   string `json:"strategy"`
		}
		err := json.NewDecoder(r.Body).Decode(&request)
		c.Assert(err, qt.IsNil)
		c.Assert(request.Strategy, qt.Equals, "parallel")
		w.WriteHeader(200)
		out := `{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": "2021-01-14T10:19:23.000Z", "deployment": { "state": "queued" }, "number": 1337}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	_, err = client.DeployRequests.Deploy(ctx, &PerformDeployRequest{
		Organization: "test-organization",
		Database:     "test-database",
		Number:       1337,
		Strategy:     "parallel",
	})
	c.Assert(err, qt.IsNil)
}

func TestDeployRequests_DeployOmitsEmptyStrategy(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		c.Assert(err, qt.IsNil)
		c.Assert(strings.Contains(string(raw), "strategy"), qt.IsFalse)
		w.WriteHeader(200)
		out := `{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": "2021-01-14T10:19:23.000Z", "deployment": { "state": "queued" }, "number": 1337}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	_, err = client.DeployRequests.Deploy(ctx, &PerformDeployRequest{
		Organization: "test-organization",
		Database:     "test-database",
		Number:       1337,
	})
	c.Assert(err, qt.IsNil)
}

func TestDeployRequests_CancelDeploy(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": "2021-01-14T10:19:23.000Z", "deployment": { "state": "pending" }, "number": 1337}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	dr, err := client.DeployRequests.CancelDeploy(ctx, &CancelDeployRequestRequest{
		Organization: "test-organization",
		Database:     "test-database",
		Number:       1337,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := &DeployRequest{
		ID:     "test-deploy-request-id",
		Branch: "development",
		Deployment: &Deployment{
			State: "pending",
		},
		IntoBranch: "some-branch",
		Number:     1337,
		Notes:      "",
		CreatedAt:  testTime,
		UpdatedAt:  testTime,
		ClosedAt:   &testTime,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(dr, qt.DeepEquals, want)
}

func TestDeployRequests_ForceCutover(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/test-organization/databases/test-database/deploy-requests/1337/force-cutover")
		w.WriteHeader(200)
		out := `{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": null, "deployment": { "state": "in_progress_cutover", "force_cutover_requested_at": "2021-01-14T10:19:23.000Z" }, "number": 1337}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	dr, err := client.DeployRequests.ForceCutover(ctx, &ForceCutoverDeployRequestRequest{
		Organization: "test-organization",
		Database:     "test-database",
		Number:       1337,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := &DeployRequest{
		ID:     "test-deploy-request-id",
		Branch: "development",
		Deployment: &Deployment{
			State:                   "in_progress_cutover",
			ForceCutoverRequestedAt: &testTime,
		},
		IntoBranch: "some-branch",
		Number:     1337,
		Notes:      "",
		CreatedAt:  testTime,
		UpdatedAt:  testTime,
		ClosedAt:   nil,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(dr, qt.DeepEquals, want)
}

func TestDeployRequests_Close(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": "2021-01-14T10:19:23.000Z", "deployment": { "state": "pending" }, "number": 1337}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	dr, err := client.DeployRequests.CloseDeploy(ctx, &CloseDeployRequestRequest{
		Organization: "test-organization",
		Database:     "test-database",
		Number:       1337,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := &DeployRequest{
		ID:     "test-deploy-request-id",
		Branch: "development",
		Deployment: &Deployment{
			State: "pending",
		},
		IntoBranch: "some-branch",
		Number:     1337,
		Notes:      "",
		CreatedAt:  testTime,
		UpdatedAt:  testTime,
		ClosedAt:   &testTime,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(dr, qt.DeepEquals, want)
}

func TestDeployRequests_Create(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id": "test-deploy-request-id", "number": 1337, "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": "2021-01-14T10:19:23.000Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	requests, err := client.DeployRequests.Create(ctx, &CreateDeployRequestRequest{
		Organization:     testOrg,
		Database:         testDatabase,
		Notes:            "",
		AutoDeleteBranch: true,
		AutoCutover:      false,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := &DeployRequest{
		ID:         "test-deploy-request-id",
		Number:     1337,
		Branch:     "development",
		IntoBranch: "some-branch",
		Notes:      "",
		CreatedAt:  testTime,
		UpdatedAt:  testTime,
		ClosedAt:   &testTime,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(requests, qt.DeepEquals, want)
}

func TestDeployRequests_Review(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id": "test-review-id","type": "DeployRequestReview","body": "test body","html_body": "","state": "approved","created_at": "2021-01-14T10:19:23.000Z","updated_at": "2021-01-14T10:19:23.000Z"}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	requests, err := client.DeployRequests.CreateReview(ctx, &ReviewDeployRequestRequest{
		Organization: testOrg,
		Database:     testDatabase,
		CommentText:  "test body",
		ReviewAction: ReviewApprove,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := &DeployRequestReview{
		ID:        "test-review-id",
		Body:      "test body",
		State:     "approved",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(requests, qt.DeepEquals, want)
}

func TestDeployRequests_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"data": [{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": "2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	requests, err := client.DeployRequests.List(ctx, &ListDeployRequestsRequest{
		Organization: testOrg,
		Database:     testDatabase,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := []*DeployRequest{
		{
			ID:         "test-deploy-request-id",
			Branch:     "development",
			IntoBranch: "some-branch",
			Notes:      "",
			CreatedAt:  testTime,
			UpdatedAt:  testTime,
			ClosedAt:   &testTime,
		},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(requests, qt.DeepEquals, want)
}

func TestDeployRequests_ListQueryParams(t *testing.T) {
	c := qt.New(t)

	var receivedQueryParams url.Values

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQueryParams = r.URL.Query()

		w.WriteHeader(200)
		out := `{"data": [{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": "2021-01-14T10:19:23.000Z"}]}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	requests, err := client.DeployRequests.List(ctx, &ListDeployRequestsRequest{
		Organization: testOrg,
		Database:     testDatabase,
		State:        "closed",
		Branch:       "dev",
		IntoBranch:   "main",
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := []*DeployRequest{
		{
			ID:         "test-deploy-request-id",
			Branch:     "development",
			IntoBranch: "some-branch",
			Notes:      "",
			CreatedAt:  testTime,
			UpdatedAt:  testTime,
			ClosedAt:   &testTime,
		},
	}

	c.Assert(err, qt.IsNil)
	c.Assert(requests, qt.DeepEquals, want)

	// Assert the expected query parameters
	c.Assert(receivedQueryParams.Get("state"), qt.Equals, "closed")
	c.Assert(receivedQueryParams.Get("branch"), qt.Equals, "dev")
	c.Assert(receivedQueryParams.Get("into_branch"), qt.Equals, "main")
}

func TestDeployRequests_SkipRevertDeploy(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": "2021-01-14T10:19:23.000Z", "deployment": { "state": "complete" }, "number": 1337}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	dr, err := client.DeployRequests.SkipRevertDeploy(ctx, &SkipRevertDeployRequestRequest{
		Organization: "test-organization",
		Database:     "test-database",
		Number:       1337,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := &DeployRequest{
		ID:     "test-deploy-request-id",
		Branch: "development",
		Deployment: &Deployment{
			State: "complete",
		},
		IntoBranch: "some-branch",
		Number:     1337,
		Notes:      "",
		CreatedAt:  testTime,
		UpdatedAt:  testTime,
		ClosedAt:   &testTime,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(dr, qt.DeepEquals, want)
}

func TestDeployRequests_RevertDeploy(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{"id": "test-deploy-request-id", "branch": "development", "into_branch": "some-branch", "notes": "", "created_at": "2021-01-14T10:19:23.000Z", "updated_at": "2021-01-14T10:19:23.000Z", "closed_at": "2021-01-14T10:19:23.000Z", "deployment": { "state": "complete_revert" }, "number": 1337}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	dr, err := client.DeployRequests.RevertDeploy(ctx, &RevertDeployRequestRequest{
		Organization: "test-organization",
		Database:     "test-database",
		Number:       1337,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := &DeployRequest{
		ID:     "test-deploy-request-id",
		Branch: "development",
		Deployment: &Deployment{
			State: "complete_revert",
		},
		IntoBranch: "some-branch",
		Number:     1337,
		Notes:      "",
		CreatedAt:  testTime,
		UpdatedAt:  testTime,
		ClosedAt:   &testTime,
	}

	c.Assert(err, qt.IsNil)
	c.Assert(dr, qt.DeepEquals, want)
}

func TestDeployRequests_DeployOperations(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		out := `{
		"type":"list",
		"current_page":1,
		"data":[
			{
				"id":"test-operation-id",
				"type":"DeployOperation",
				"state":"pending",
				"keyspace_name":"treats",
				"table_name":"ice_creams",
				"operation_name":"CREATE",
				"created_at":"2021-01-14T10:19:23.000Z",
				"updated_at":"2021-01-14T10:19:23.000Z"
			}
		]
	}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	ctx := context.Background()

	do, err := client.DeployRequests.GetDeployOperations(ctx, &GetDeployOperationsRequest{
		Organization: "test-organization",
		Database:     "test-database",
		Number:       1337,
	})

	testTime := time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC)

	want := []*DeployOperation{{
		ID:                 "test-operation-id",
		State:              "pending",
		Table:              "ice_creams",
		Keyspace:           "treats",
		Operation:          "CREATE",
		ETASeconds:         0,
		ProgressPercentage: 0,
		CreatedAt:          testTime,
		UpdatedAt:          testTime,
	}}
	c.Assert(err, qt.IsNil)
	c.Assert(do, qt.DeepEquals, want)
}
