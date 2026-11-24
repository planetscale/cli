package pgbouncer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func testRestart(state string) *ps.PostgresBouncerRestart {
	return &ps.PostgresBouncerRestart{
		ID:        "restart-1",
		State:     state,
		Target:    "dedicated",
		Bouncer:   "read-pool",
		CreatedAt: time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		UpdatedAt: time.Date(2021, 1, 14, 10, 19, 23, 0, time.UTC),
		Actor:     ps.Actor{ID: "user-1", Name: "Alice"},
	}
}

func postgresDBService(db string) *mock.DatabaseService {
	return &mock.DatabaseService{
		GetFn: func(ctx context.Context, req *ps.GetDatabaseRequest) (*ps.Database, error) {
			return postgresDB(db), nil
		},
	}
}

func TestPgBouncer_RestartCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	org := "planetscale"
	db := "mydb"
	branch := "main"
	name := "read-pool"
	restart := testRestart(ps.PostgresBouncerRestartStatePending)

	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branch)
			c.Assert(req.Bouncer, qt.Equals, name)
			return restart, nil
		},
	}

	cmd := RestartCmd(testHelper(org, postgresDBService(db), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{db, branch, name, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateRestartFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &PgBouncerRestart{orig: restart})
}

func TestPgBouncer_RestartCmd_RequiresForceInJSON(t *testing.T) {
	c := qt.New(t)

	svc := &mock.PostgresBouncersService{}
	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "read-pool"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*run with --force.*`)
	c.Assert(svc.CreateRestartFnInvoked, qt.IsFalse)
}

func TestPgBouncer_RestartCmd_NotFound(t *testing.T) {
	c := qt.New(t)

	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "missing", "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*does not exist.*`)
}

func TestPgBouncer_RestartCmd_FailedJSONEnvelope(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	restart := testRestart(ps.PostgresBouncerRestartStateFailed)
	restart.Error = "pod useast1b did not become ready"

	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return restart, nil
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "--force"})
	err := cmd.Execute()

	var cmdErr *cmdutil.Error
	c.Assert(errors.As(err, &cmdErr), qt.IsTrue)
	c.Assert(cmdErr.Handled, qt.IsTrue)
	c.Assert(cmdErr.ExitCode, qt.Equals, cmdutil.FatalErrExitCode)
	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"status": "error",
		"error":  "restart restart-1 of PgBouncer read-pool failed: pod useast1b did not become ready",
		"id":     "restart-1",
		"state":  "failed",
		"issues": []any{map[string]any{
			"code":    "PGBOUNCER_RESTART_FAILED",
			"message": "restart restart-1 of PgBouncer read-pool failed: pod useast1b did not become ready",
		}},
		"next_steps": []any{"pscale pgbouncer restart status mydb main read-pool restart-1 --org planetscale --format json"},
	})
}

func TestPgBouncer_RestartCmd_FailedHuman(t *testing.T) {
	c := qt.New(t)

	restart := testRestart(ps.PostgresBouncerRestartStateFailed)
	restart.Error = "pod useast1b did not become ready"

	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return restart, nil
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.Human, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `restart restart-1 of PgBouncer read-pool failed: pod useast1b did not become ready`)
}

func TestPgBouncer_RestartCmd_WaitCanceled(t *testing.T) {
	c := qt.New(t)
	restartPollInterval = time.Millisecond
	c.Cleanup(func() { restartPollInterval = 5 * time.Second })

	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return testRestart(ps.PostgresBouncerRestartStatePending), nil
		},
		GetRestartFn: func(ctx context.Context, req *ps.GetPostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return testRestart(ps.PostgresBouncerRestartStateCanceled), nil
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.Human, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "--force", "--wait"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `restart restart-1 of PgBouncer read-pool was canceled`)
}

func TestPgBouncer_RestartCmd_WaitSucceeds(t *testing.T) {
	c := qt.New(t)
	restartPollInterval = time.Millisecond
	c.Cleanup(func() { restartPollInterval = 5 * time.Second })

	var buf bytes.Buffer
	states := []string{"", ps.PostgresBouncerRestartStateRunning, ps.PostgresBouncerRestartStateSucceeded}
	polls := 0
	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return testRestart(ps.PostgresBouncerRestartStatePending), nil
		},
		GetRestartFn: func(ctx context.Context, req *ps.GetPostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			c.Assert(req.ID, qt.Equals, "restart-1")
			state := states[polls]
			polls++
			if state == "" {
				// a blip between polls must not end the wait
				return nil, &ps.Error{Code: ps.ErrInternal, HTTPStatus: http.StatusBadGateway}
			}
			return testRestart(state), nil
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "--force", "--wait"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(polls, qt.Equals, 3)
	c.Assert(buf.String(), qt.JSONEquals, &PgBouncerRestart{orig: testRestart(ps.PostgresBouncerRestartStateSucceeded)})
}

func TestPgBouncer_RestartCmd_WaitStopsOnPermanentError(t *testing.T) {
	c := qt.New(t)
	restartPollInterval = time.Millisecond
	c.Cleanup(func() { restartPollInterval = 5 * time.Second })

	polls := 0
	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return testRestart(ps.PostgresBouncerRestartStatePending), nil
		},
		GetRestartFn: func(ctx context.Context, req *ps.GetPostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			polls++
			return nil, &ps.Error{Code: ps.ErrPermission}
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.Human, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "--force", "--wait"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s)stopped waiting for restart restart-1: .*'pscale pgbouncer restart status mydb main read-pool restart-1 --org planetscale'`)
	c.Assert(polls, qt.Equals, 1)
}

func TestPgBouncer_RestartCmd_WaitErrorKeepsRestartID(t *testing.T) {
	c := qt.New(t)
	restartPollInterval = time.Millisecond
	c.Cleanup(func() { restartPollInterval = 5 * time.Second })

	var buf bytes.Buffer
	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return testRestart(ps.PostgresBouncerRestartStatePending), nil
		},
		GetRestartFn: func(ctx context.Context, req *ps.GetPostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return nil, &ps.Error{Code: ps.ErrPermission, HTTPStatus: http.StatusForbidden}
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "--force", "--wait"})
	err := cmd.Execute()

	var cmdErr *cmdutil.Error
	c.Assert(errors.As(err, &cmdErr), qt.IsTrue)
	c.Assert(cmdErr.Handled, qt.IsTrue)

	var out struct {
		ID        string   `json:"id"`
		State     string   `json:"state"`
		NextSteps []string `json:"next_steps"`
	}
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out.ID, qt.Equals, "restart-1")
	c.Assert(out.State, qt.Equals, "pending")
	c.Assert(out.NextSteps, qt.Contains, "pscale pgbouncer restart status mydb main read-pool restart-1 --org planetscale --format json")
}

func TestPgBouncer_RestartCmd_RejectsExtraArgs(t *testing.T) {
	c := qt.New(t)

	svc := &mock.PostgresBouncersService{}
	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.JSON, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "restart-1", "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(svc.CreateRestartFnInvoked, qt.IsFalse)
}

func TestTransientRestartError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"gateway html", http.StatusBadGateway, "<html>Bad Gateway</html>", true},
		{"unavailable empty", http.StatusServiceUnavailable, "", true},
		{"internal api error", http.StatusInternalServerError, `{"code":"internal","message":"boom"}`, true},
		{"unprocessable", http.StatusUnprocessableEntity, `{"code":"unprocessable","message":"try again"}`, true},
		{"expired token", http.StatusUnauthorized, `{"error":"invalid_token","error_description":"expired"}`, false},
		{"forbidden", http.StatusForbidden, `{"code":"forbidden","message":"no"}`, false},
		{"not found", http.StatusNotFound, `{"code":"not_found","message":"gone"}`, false},
		{"undecodable success", http.StatusOK, `{"id": 1}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := qt.New(t)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer ts.Close()

			client, err := ps.NewClient(ps.WithBaseURL(ts.URL))
			c.Assert(err, qt.IsNil)
			_, err = client.PostgresBouncers.GetRestart(context.Background(), &ps.GetPostgresBouncerRestartRequest{
				Organization: "org", Database: "db", Branch: "main", Bouncer: "read-pool", ID: "restart-1",
			})
			c.Assert(err, qt.IsNotNil)
			c.Assert(transientRestartError(err), qt.Equals, tt.want)
		})
	}
}

func TestPgBouncer_RestartCmd_WaitStopsOnUndecodableResponse(t *testing.T) {
	c := qt.New(t)
	restartPollInterval = time.Millisecond
	c.Cleanup(func() { restartPollInterval = 5 * time.Second })

	polls := 0
	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return testRestart(ps.PostgresBouncerRestartStatePending), nil
		},
		GetRestartFn: func(ctx context.Context, req *ps.GetPostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			polls++
			return nil, errors.New("json: cannot unmarshal number into Go struct field .state of type string")
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.Human, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "--force", "--wait"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(polls, qt.Equals, 1)
}

func TestPgBouncer_RestartCmd_WaitTimeout(t *testing.T) {
	c := qt.New(t)
	restartPollInterval = time.Millisecond
	c.Cleanup(func() { restartPollInterval = 5 * time.Second })

	polls := 0
	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return testRestart(ps.PostgresBouncerRestartStatePending), nil
		},
		GetRestartFn: func(ctx context.Context, req *ps.GetPostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			polls++
			if polls > 1 {
				return nil, &ps.Error{Code: ps.ErrInternal, HTTPStatus: http.StatusServiceUnavailable}
			}
			return testRestart(ps.PostgresBouncerRestartStateRunning), nil
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.Human, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "--force", "--wait", "--wait-timeout", "20ms"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s)restart restart-1 did not finish within 20ms \(current state: running\); last poll failed: .*'pscale pgbouncer restart status mydb main read-pool restart-1 --org planetscale'`)
}

func TestPgBouncer_RestartCmd_WaitTimeoutJSON(t *testing.T) {
	c := qt.New(t)
	restartPollInterval = time.Millisecond
	c.Cleanup(func() { restartPollInterval = 5 * time.Second })

	var buf bytes.Buffer
	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return testRestart(ps.PostgresBouncerRestartStatePending), nil
		},
		GetRestartFn: func(ctx context.Context, req *ps.GetPostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return testRestart(ps.PostgresBouncerRestartStateRunning), nil
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "--force", "--wait", "--wait-timeout", "20ms"})
	err := cmd.Execute()

	var cmdErr *cmdutil.Error
	c.Assert(errors.As(err, &cmdErr), qt.IsTrue)
	c.Assert(cmdErr.Handled, qt.IsTrue)
	c.Assert(cmdErr.ExitCode, qt.Equals, cmdutil.ActionRequestedExitCode)
	c.Assert(buf.String(), qt.JSONEquals, map[string]any{
		"status": "action_required",
		"error":  "restart restart-1 did not finish within 20ms (current state: running)",
		"id":     "restart-1",
		"state":  "running",
		"issues": []any{map[string]any{
			"code":    "PGBOUNCER_RESTART_WAIT_TIMEOUT",
			"message": "restart restart-1 did not finish within 20ms (current state: running)",
		}},
		"next_steps": []any{"pscale pgbouncer restart status mydb main read-pool restart-1 --org planetscale --format json"},
	})
}

func TestPgBouncer_RestartCmd_Primary(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	restart := testRestart(ps.PostgresBouncerRestartStatePending)
	restart.Target = "local"
	restart.Bouncer = ps.PostgresBouncerLocalName

	svc := &mock.PostgresBouncersService{
		CreateRestartFn: func(ctx context.Context, req *ps.CreatePostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			c.Assert(req.Bouncer, qt.Equals, ps.PostgresBouncerLocalName)
			return restart, nil
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"mydb", "main", "primary", "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateRestartFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &PgBouncerRestart{orig: restart})
}

func TestPgBouncer_RestartStatusCmd_Latest(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	restart := testRestart(ps.PostgresBouncerRestartStateRunning)

	svc := &mock.PostgresBouncersService{
		ListRestartsFn: func(ctx context.Context, req *ps.ListPostgresBouncerRestartsRequest, opts ...ps.ListOption) ([]*ps.PostgresBouncerRestart, error) {
			c.Assert(req.Bouncer, qt.Equals, "primary")
			return []*ps.PostgresBouncerRestart{restart}, nil
		},
	}

	cmd := RestartCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"status", "mydb", "main", "primary"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListRestartsFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &PgBouncerRestart{orig: restart})
}

func TestPgBouncer_RestartStatusCmd_ByID(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	restart := testRestart(ps.PostgresBouncerRestartStateSucceeded)

	svc := &mock.PostgresBouncersService{
		GetRestartFn: func(ctx context.Context, req *ps.GetPostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			c.Assert(req.Bouncer, qt.Equals, "read-pool")
			c.Assert(req.ID, qt.Equals, "restart-1")
			return restart, nil
		},
	}

	cmd := RestartStatusCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "restart-1"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetRestartFnInvoked, qt.IsTrue)
	c.Assert(svc.ListRestartsFnInvoked, qt.IsFalse)
	c.Assert(buf.String(), qt.JSONEquals, &PgBouncerRestart{orig: restart})
}

func TestPgBouncer_RestartStatusCmd_ByIDNotFound(t *testing.T) {
	c := qt.New(t)

	svc := &mock.PostgresBouncersService{
		GetRestartFn: func(ctx context.Context, req *ps.GetPostgresBouncerRestartRequest) (*ps.PostgresBouncerRestart, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	cmd := RestartStatusCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.Human, &bytes.Buffer{}))
	cmd.SetArgs([]string{"mydb", "main", "read-pool", "missing"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*restart .*missing.* of PgBouncer .*read-pool.* does not exist.*`)
}

func TestPgBouncer_RestartStatusCmd_None(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	svc := &mock.PostgresBouncersService{
		ListRestartsFn: func(ctx context.Context, req *ps.ListPostgresBouncerRestartsRequest, opts ...ps.ListOption) ([]*ps.PostgresBouncerRestart, error) {
			return nil, nil
		},
	}

	cmd := RestartStatusCmd(testHelper("planetscale", postgresDBService("mydb"), svc, printer.JSON, &buf))
	cmd.SetArgs([]string{"mydb", "main", "read-pool"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(buf.String(), qt.JSONEquals, []*PgBouncerRestart{})
}
