package deployrequest

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	"github.com/planetscale/cli/internal/printer"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

func TestDeployRequest_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	var number uint64 = 10

	svc := &mock.DeployRequestsService{
		GetFn: func(ctx context.Context, req *ps.GetDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, number)

			return &ps.DeployRequest{Number: number}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10)})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)

	res := &ps.DeployRequest{Number: number}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDeployRequest_ShowBranchName(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "planetscale"
	number := uint64(10)
	branchName := "dev"

	svc := &mock.DeployRequestsService{
		GetFn: func(ctx context.Context, req *ps.GetDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, number)

			return &ps.DeployRequest{Number: number}, nil
		},
		ListFn: func(ctx context.Context, req *ps.ListDeployRequestsRequest) ([]*ps.DeployRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Branch, qt.Equals, branchName)

			return []*ps.DeployRequest{
				{
					Number: number,
				},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, branchName})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)

	res := &ps.DeployRequest{Number: number}
	c.Assert(buf.String(), qt.JSONEquals, res)
}

func TestDeployRequest_ShowTimestampBug(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.Human // Use human format to test table output
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "testdb"
	var number uint64 = 47

	// Create timestamps for testing - make them different to verify correct field assignment
	createdAt := time.Date(2025, 6, 23, 22, 46, 42, 348000000, time.UTC) // Base timestamp
	updatedAt := time.Date(2025, 6, 23, 22, 52, 34, 76000000, time.UTC)  // 6 minutes later
	startedAt := time.Date(2025, 6, 23, 22, 48, 52, 553000000, time.UTC) // 2 minutes after created
	queuedAt := time.Date(2025, 6, 23, 22, 48, 38, 809000000, time.UTC)  // Just before started

	svc := &mock.DeployRequestsService{
		GetFn: func(ctx context.Context, req *ps.GetDeployRequestRequest) (*ps.DeployRequest, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Number, qt.Equals, number)

			return &ps.DeployRequest{
				ID:         "abcd1234efgh",
				Number:     number,
				Branch:     "feature-branch-2025", // String, not timestamp
				IntoBranch: "main",                // String, not timestamp
				Approved:   true,
				State:      "open",
				CreatedAt:  createdAt,
				UpdatedAt:  updatedAt,
				Deployment: &ps.Deployment{
					ID:                 "deploy5678wxyz",
					State:              "in_progress",
					Deployable:         true,
					InstantDDLEligible: false,
					StartedAt:          &startedAt,
					QueuedAt:           &queuedAt,
					FinishedAt:         nil, // This should show as empty, not showing incorrect timestamp
				},
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config: &config.Config{
			Organization: org,
		},
		Client: func() (*ps.Client, error) {
			return &ps.Client{
				DeployRequests: svc,
			}, nil
		},
	}

	cmd := ShowCmd(ch)
	cmd.SetArgs([]string{db, strconv.FormatUint(number, 10)})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)

	output := buf.String()

	// Debug: Print the actual output to see what we get
	t.Logf("Table output:\n%s", output)

	// Debug: Print the actual struct values being passed to tableprinter
	dr, _ := svc.GetFn(context.Background(), &ps.GetDeployRequestRequest{
		Organization: org,
		Database:     db,
		Number:       number,
	})
	converted := toDeployRequest(dr)
	t.Logf("Struct values - CreatedAt: %v, UpdatedAt: %v, FinishedAt: %v, StartedAt: %v, QueuedAt: %v",
		converted.CreatedAt, converted.UpdatedAt, converted.Deployment.FinishedAt, converted.Deployment.StartedAt, converted.Deployment.QueuedAt)

	// Test the specific bug: FINISHED AT should be empty when deployment.finished_at is nil
	// Look for the FINISHED AT column in the table output
	lines := strings.Split(output, "\n")
	headerLine := ""
	dataLine := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.Contains(line, "---") {
			if headerLine == "" {
				headerLine = line
			} else if dataLine == "" {
				dataLine = line
				break
			}
		}
	}

	c.Assert(headerLine, qt.Not(qt.Equals), "", qt.Commentf("Could not find header line"))
	c.Assert(dataLine, qt.Not(qt.Equals), "", qt.Commentf("Could not find data line"))

	// Find the FINISHED AT column position
	finishedAtPos := strings.Index(headerLine, "FINISHED AT")
	c.Assert(finishedAtPos, qt.Not(qt.Equals), -1, qt.Commentf("Could not find FINISHED AT column in header"))

	// Find the next column after FINISHED AT to know where this column ends
	remainingHeader := headerLine[finishedAtPos+len("FINISHED AT"):]
	nextColumnMatch := strings.Fields(remainingHeader)
	var finishedAtEndPos int
	if len(nextColumnMatch) > 0 {
		nextColumnPos := strings.Index(remainingHeader, nextColumnMatch[0])
		finishedAtEndPos = finishedAtPos + len("FINISHED AT") + nextColumnPos
	} else {
		finishedAtEndPos = len(headerLine)
	}

	// Extract the FINISHED AT column value from the data line
	if finishedAtEndPos <= len(dataLine) {
		finishedAtValue := strings.TrimSpace(dataLine[finishedAtPos:finishedAtEndPos])

		// The bug: FINISHED AT should be empty since deployment.finished_at is nil
		// If it contains "ago" or any timestamp value, that's the bug
		if finishedAtValue != "" && strings.Contains(finishedAtValue, "ago") {
			c.Errorf("FINISHED AT column shows '%s' but deployment.finished_at is nil - should be empty", finishedAtValue)
		}
	}
}
