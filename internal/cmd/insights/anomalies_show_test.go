package insights

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
)

func TestInsights_AnomaliesShowCmd(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		GetAnomalyFn: func(ctx context.Context, req *ps.GetAnomalyRequest) (*ps.Anomaly, error) {
			c.Assert(req.Organization, qt.Equals, "planetscale")
			c.Assert(req.Database, qt.Equals, "mydb")
			c.Assert(req.Branch, qt.Equals, "main")
			c.Assert(req.AnomalyID, qt.Equals, "anomaly-123")
			return &ps.Anomaly{
				ID:                 "anomaly-123",
				PeriodStart:        time.Date(2026, 8, 11, 18, 0, 0, 0, time.UTC),
				MinutesInViolation: 12,
				Correlations: []ps.Correlation{{
					ID:            "corr-1",
					R:             0.94,
					Fingerprint:   "b129e8fa",
					Keyspace:      "main",
					NormalizedSQL: "select * from users where id = ?",
					TabletType:    "primary",
				}},
			}, nil
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := AnomaliesShowCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "anomaly-123"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetAnomalyFnInvoked, qt.IsTrue)

	var out map[string]any
	c.Assert(json.Unmarshal(buf.Bytes(), &out), qt.IsNil)
	c.Assert(out["id"], qt.Equals, "anomaly-123")
	c.Assert(out["correlations"], qt.HasLen, 1)
}

func TestInsights_AnomaliesShowCmd_NotFound(t *testing.T) {
	c := qt.New(t)

	svc := &mock.QueryInsightsService{
		GetAnomalyFn: func(ctx context.Context, req *ps.GetAnomalyRequest) (*ps.Anomaly, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{QueryInsights: svc})

	cmd := AnomaliesShowCmd(ch)
	cmd.SetArgs([]string{"mydb", "main", "anomaly-123"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "does not exist")
}

func TestInsights_AnomaliesCmd_HasShowSubcommand(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	ch := testHelper(&buf, printer.JSON, &ps.Client{})

	cmd, _, err := AnomaliesCmd(ch).Find([]string{"show"})
	c.Assert(err, qt.IsNil)
	c.Assert(cmd.Name(), qt.Equals, "show")
}
