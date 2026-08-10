package backup

import (
	"bytes"
	"context"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestBackupPolicy_ListCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	policies := []*ps.BackupPolicy{{
		ID:             "policy-1",
		Name:           "Production daily",
		Target:         "production",
		RetentionValue: 2,
		RetentionUnit:  "day",
		FrequencyValue: 12,
		FrequencyUnit:  "hour",
		ScheduleTime:   "09:10",
		Required:       true,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}}

	svc := &mock.BackupPoliciesService{
		ListFn: func(ctx context.Context, req *ps.ListBackupPoliciesRequest, opts ...ps.ListOption) ([]*ps.BackupPolicy, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			return policies, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{BackupPolicies: svc}, nil
		},
	}

	cmd := PolicyListCmd(ch)
	cmd.SetArgs([]string{db})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.ListFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, []*BackupPolicy{{orig: policies[0]}})
}

func TestBackupPolicy_ShowCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	policyID := "policy-1"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	policy := &ps.BackupPolicy{
		ID:             policyID,
		Name:           "Production daily",
		Target:         "production",
		RetentionValue: 2,
		RetentionUnit:  "day",
		FrequencyValue: 12,
		FrequencyUnit:  "hour",
		ScheduleTime:   "09:10",
		Required:       true,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}

	svc := &mock.BackupPoliciesService{
		GetFn: func(ctx context.Context, req *ps.GetBackupPolicyRequest) (*ps.BackupPolicy, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Policy, qt.Equals, policyID)
			return policy, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{BackupPolicies: svc}, nil
		},
	}

	cmd := PolicyShowCmd(ch)
	cmd.SetArgs([]string{db, policyID})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &BackupPolicy{orig: policy})
}

func TestBackupPolicy_CreateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	day := 1
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	policy := &ps.BackupPolicy{
		ID:             "policy-2",
		Name:           "Weekly prod",
		Target:         "production",
		RetentionValue: 7,
		RetentionUnit:  "day",
		FrequencyValue: 1,
		FrequencyUnit:  "week",
		ScheduleTime:   "03:00",
		ScheduleDay:    &day,
		Required:       false,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}

	svc := &mock.BackupPoliciesService{
		CreateFn: func(ctx context.Context, req *ps.CreateBackupPolicyRequest) (*ps.BackupPolicy, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Name, qt.Equals, "Weekly prod")
			c.Assert(req.Target, qt.Equals, "production")
			c.Assert(req.RetentionValue, qt.Equals, 7)
			c.Assert(req.RetentionUnit, qt.Equals, "day")
			c.Assert(req.FrequencyValue, qt.Equals, 1)
			c.Assert(req.FrequencyUnit, qt.Equals, "week")
			c.Assert(req.ScheduleTime, qt.Equals, "03:00")
			c.Assert(req.ScheduleDay, qt.IsNotNil)
			c.Assert(*req.ScheduleDay, qt.Equals, 1)
			return policy, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{BackupPolicies: svc}, nil
		},
	}

	cmd := PolicyCreateCmd(ch)
	cmd.SetArgs([]string{
		db,
		"--name", "Weekly prod",
		"--target", "production",
		"--retention-value", "7",
		"--retention-unit", "day",
		"--frequency-value", "1",
		"--frequency-unit", "week",
		"--schedule-time", "03:00",
		"--schedule-day", "1",
	})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.CreateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &BackupPolicy{orig: policy})
}

func TestBackupPolicy_UpdateCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	policyID := "policy-1"
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	policy := &ps.BackupPolicy{
		ID:             policyID,
		Name:           "Production daily",
		Target:         "production",
		RetentionValue: 14,
		RetentionUnit:  "day",
		FrequencyValue: 12,
		FrequencyUnit:  "hour",
		ScheduleTime:   "09:10",
		Required:       true,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}

	svc := &mock.BackupPoliciesService{
		UpdateFn: func(ctx context.Context, req *ps.UpdateBackupPolicyRequest) (*ps.BackupPolicy, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Policy, qt.Equals, policyID)
			c.Assert(req.RetentionValue, qt.IsNotNil)
			c.Assert(*req.RetentionValue, qt.Equals, 14)
			c.Assert(req.RetentionUnit, qt.IsNotNil)
			c.Assert(*req.RetentionUnit, qt.Equals, "day")
			c.Assert(req.Target, qt.IsNil)
			return policy, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{BackupPolicies: svc}, nil
		},
	}

	cmd := PolicyUpdateCmd(ch)
	cmd.SetArgs([]string{db, policyID, "--retention-value", "14", "--retention-unit", "day"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.UpdateFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, &BackupPolicy{orig: policy})
}

func TestBackupPolicy_UpdateCmd_NoFlags(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	svc := &mock.BackupPoliciesService{}
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{Organization: "planetscale"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{BackupPolicies: svc}, nil
		},
	}

	cmd := PolicyUpdateCmd(ch)
	cmd.SetArgs([]string{"mydb", "policy-1"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, "at least one policy flag must be provided")
	c.Assert(svc.UpdateFnInvoked, qt.IsFalse)
}

func TestBackupPolicy_DeleteCmd(t *testing.T) {
	c := qt.New(t)

	var buf bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&buf)

	org := "planetscale"
	db := "mydb"
	policyID := "policy-2"

	svc := &mock.BackupPoliciesService{
		GetFn: func(ctx context.Context, req *ps.GetBackupPolicyRequest) (*ps.BackupPolicy, error) {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Policy, qt.Equals, policyID)
			return &ps.BackupPolicy{ID: policyID, Required: false}, nil
		},
		DeleteFn: func(ctx context.Context, req *ps.DeleteBackupPolicyRequest) error {
			c.Assert(req.Organization, qt.Equals, org)
			c.Assert(req.Database, qt.Equals, db)
			c.Assert(req.Policy, qt.Equals, policyID)
			return nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{BackupPolicies: svc}, nil
		},
	}

	cmd := PolicyDeleteCmd(ch)
	cmd.SetArgs([]string{db, policyID, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.IsNil)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.DeleteFnInvoked, qt.IsTrue)
	c.Assert(buf.String(), qt.JSONEquals, map[string]string{
		"result": "backup policy deleted",
		"policy": policyID,
	})
}

func TestBackupPolicy_DeleteCmd_RejectsRequiredDefault(t *testing.T) {
	c := qt.New(t)

	format := printer.JSON
	org := "planetscale"
	db := "mydb"
	policyID := "policy-1"

	svc := &mock.BackupPoliciesService{
		GetFn: func(ctx context.Context, req *ps.GetBackupPolicyRequest) (*ps.BackupPolicy, error) {
			return &ps.BackupPolicy{
				ID:       policyID,
				Name:     "Production daily",
				Required: true,
			}, nil
		},
	}

	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{Organization: org},
		Client: func() (*ps.Client, error) {
			return &ps.Client{BackupPolicies: svc}, nil
		},
	}

	cmd := PolicyDeleteCmd(ch)
	cmd.SetArgs([]string{db, policyID, "--force"})
	err := cmd.Execute()

	c.Assert(err, qt.ErrorMatches, `(?s).*required default policy and cannot be deleted.*`)
	c.Assert(svc.GetFnInvoked, qt.IsTrue)
	c.Assert(svc.DeleteFnInvoked, qt.IsFalse)
}
