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

func TestBackupPolicies_List(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/backup-policies")
		w.WriteHeader(200)
		out := `{
			"data":[{
				"id":"policy-1",
				"display_name":"Backup policy Production daily",
				"name":"Production daily",
				"target":"production",
				"retention_value":2,
				"retention_unit":"day",
				"frequency_value":12,
				"frequency_unit":"hour",
				"schedule_time":"09:10",
				"schedule_day":null,
				"schedule_week":null,
				"required":true,
				"created_at":"2021-01-14T10:19:23.000Z",
				"updated_at":"2021-01-14T10:19:23.000Z",
				"last_ran_at":null,
				"next_run_at":"2021-01-15T03:23:00.000Z"
			}]
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	policies, err := client.BackupPolicies.List(context.Background(), &ListBackupPoliciesRequest{
		Organization: testOrg,
		Database:     "my-db",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(policies, qt.HasLen, 1)
	c.Assert(policies[0].ID, qt.Equals, "policy-1")
	c.Assert(policies[0].Target, qt.Equals, "production")
	c.Assert(policies[0].Required, qt.IsTrue)
	c.Assert(policies[0].ScheduleDay, qt.IsNil)
	c.Assert(policies[0].NextRunAt, qt.IsNotNil)
	c.Assert(policies[0].NextRunAt.Equal(time.Date(2021, time.January, 15, 3, 23, 0, 0, time.UTC)), qt.IsTrue)
}

func TestBackupPolicies_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/backup-policies/policy-1")
		w.WriteHeader(200)
		out := `{
			"id":"policy-1",
			"display_name":"Backup policy Production daily",
			"name":"Production daily",
			"target":"production",
			"retention_value":2,
			"retention_unit":"day",
			"frequency_value":12,
			"frequency_unit":"hour",
			"schedule_time":"09:10",
			"required":true,
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-14T10:19:23.000Z"
		}`
		_, err := w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	policy, err := client.BackupPolicies.Get(context.Background(), &GetBackupPolicyRequest{
		Organization: testOrg,
		Database:     "my-db",
		Policy:       "policy-1",
	})
	c.Assert(err, qt.IsNil)
	c.Assert(policy.Name, qt.Equals, "Production daily")
}

func TestBackupPolicies_Create(t *testing.T) {
	c := qt.New(t)

	day := 1
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/backup-policies")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["name"], qt.Equals, "Weekly prod")
		c.Assert(body["target"], qt.Equals, "production")
		c.Assert(body["retention_value"], qt.Equals, float64(7))
		c.Assert(body["retention_unit"], qt.Equals, "day")
		c.Assert(body["frequency_value"], qt.Equals, float64(1))
		c.Assert(body["frequency_unit"], qt.Equals, "week")
		c.Assert(body["schedule_time"], qt.Equals, "03:00")
		c.Assert(body["schedule_day"], qt.Equals, float64(1))

		w.WriteHeader(201)
		out := `{
			"id":"policy-2",
			"display_name":"Backup policy Weekly prod",
			"name":"Weekly prod",
			"target":"production",
			"retention_value":7,
			"retention_unit":"day",
			"frequency_value":1,
			"frequency_unit":"week",
			"schedule_time":"03:00",
			"schedule_day":1,
			"required":false,
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-14T10:19:23.000Z"
		}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	policy, err := client.BackupPolicies.Create(context.Background(), &CreateBackupPolicyRequest{
		Organization:   testOrg,
		Database:       "my-db",
		Name:           "Weekly prod",
		Target:         "production",
		RetentionValue: 7,
		RetentionUnit:  "day",
		FrequencyValue: 1,
		FrequencyUnit:  "week",
		ScheduleTime:   "03:00",
		ScheduleDay:    &day,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(policy.ID, qt.Equals, "policy-2")
	c.Assert(policy.Required, qt.IsFalse)
}

func TestBackupPolicies_Update(t *testing.T) {
	c := qt.New(t)

	retention := 14
	unit := "day"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPatch)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/backup-policies/policy-1")

		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		c.Assert(err, qt.IsNil)
		c.Assert(body["retention_value"], qt.Equals, float64(14))
		c.Assert(body["retention_unit"], qt.Equals, "day")
		_, ok := body["target"]
		c.Assert(ok, qt.IsFalse)

		w.WriteHeader(200)
		out := `{
			"id":"policy-1",
			"display_name":"Backup policy Production daily",
			"name":"Production daily",
			"target":"production",
			"retention_value":14,
			"retention_unit":"day",
			"frequency_value":12,
			"frequency_unit":"hour",
			"schedule_time":"09:10",
			"required":true,
			"created_at":"2021-01-14T10:19:23.000Z",
			"updated_at":"2021-01-14T10:19:23.000Z"
		}`
		_, err = w.Write([]byte(out))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	policy, err := client.BackupPolicies.Update(context.Background(), &UpdateBackupPolicyRequest{
		Organization:   testOrg,
		Database:       "my-db",
		Policy:         "policy-1",
		RetentionValue: &retention,
		RetentionUnit:  &unit,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(policy.RetentionValue, qt.Equals, 14)
}

func TestBackupPolicies_Delete(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/databases/my-db/backup-policies/policy-2")
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.BackupPolicies.Delete(context.Background(), &DeleteBackupPolicyRequest{
		Organization: testOrg,
		Database:     "my-db",
		Policy:       "policy-2",
	})
	c.Assert(err, qt.IsNil)
}
