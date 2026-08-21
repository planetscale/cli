package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/mock"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func TestUpdatePaymentMethodCreatesOpensAndPolls(t *testing.T) {
	oldInterval := paymentMethodSetupPollInterval
	oldOpenBrowser := openBrowser
	paymentMethodSetupPollInterval = time.Millisecond
	t.Cleanup(func() {
		paymentMethodSetupPollInterval = oldInterval
		openBrowser = oldOpenBrowser
	})

	var openedURL string
	openBrowser = func(_ string, url string) error {
		openedURL = url
		return nil
	}

	gets := 0
	service := &mock.BillingPaymentMethodSetupsService{
		CreateFn: func(_ context.Context, req *ps.CreateBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error) {
			if req.Organization != "my-org" {
				t.Fatalf("organization = %q", req.Organization)
			}
			return &ps.BillingPaymentMethodSetup{
				ID:          "pmsetup1",
				State:       "pending",
				CheckoutURL: "https://checkout.stripe.com/test",
			}, nil
		},
		GetFn: func(_ context.Context, req *ps.GetBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error) {
			gets++
			if req.Organization != "my-org" || req.Setup != "pmsetup1" {
				t.Fatalf("request = %#v", req)
			}
			return &ps.BillingPaymentMethodSetup{ID: "pmsetup1", State: "completed"}, nil
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&stdout)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethodSetups: service}, nil
		},
	}

	cmd := UpdatePaymentMethodCmd(ch)
	cmd.SetErr(&stderr)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if openedURL != "https://checkout.stripe.com/test" {
		t.Fatalf("opened URL = %q", openedURL)
	}
	if gets != 1 {
		t.Fatalf("GET calls = %d", gets)
	}
	if got := stdout.String(); got != "{\n  \"id\": \"pmsetup1\",\n  \"state\": \"completed\"\n}\n" {
		t.Fatalf("stdout = %q", got)
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`"browser_opened":true`)) ||
		!bytes.Contains(stderr.Bytes(), []byte(`"id":"pmsetup1"`)) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestUpdatePaymentMethodReturnsVerificationError(t *testing.T) {
	oldInterval := paymentMethodSetupPollInterval
	oldOpenBrowser := openBrowser
	paymentMethodSetupPollInterval = time.Millisecond
	openBrowser = func(_ string, _ string) error { return errors.New("headless") }
	t.Cleanup(func() {
		paymentMethodSetupPollInterval = oldInterval
		openBrowser = oldOpenBrowser
	})

	service := &mock.BillingPaymentMethodSetupsService{
		CreateFn: func(context.Context, *ps.CreateBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error) {
			return &ps.BillingPaymentMethodSetup{
				ID:          "pmsetup1",
				State:       "pending",
				CheckoutURL: "https://checkout.stripe.com/test",
			}, nil
		},
		GetFn: func(context.Context, *ps.GetBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error) {
			return &ps.BillingPaymentMethodSetup{
				ID:    "pmsetup1",
				State: "failed",
				Error: "Your card's security code is incorrect.",
			}, nil
		},
	}

	format := printer.Human
	p := printer.NewPrinter(&format)
	p.SetHumanOutput(&bytes.Buffer{})
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethodSetups: service}, nil
		},
	}

	err := UpdatePaymentMethodCmd(ch).Execute()
	if err == nil || err.Error() != "payment method setup pmsetup1 failed: Your card's security code is incorrect." {
		t.Fatalf("error = %v", err)
	}
}

func TestUpdatePaymentMethodJSONFailureEnvelope(t *testing.T) {
	oldInterval := paymentMethodSetupPollInterval
	oldOpenBrowser := openBrowser
	paymentMethodSetupPollInterval = time.Millisecond
	openBrowser = func(_ string, _ string) error { return errors.New("headless") }
	t.Cleanup(func() {
		paymentMethodSetupPollInterval = oldInterval
		openBrowser = oldOpenBrowser
	})

	service := &mock.BillingPaymentMethodSetupsService{
		CreateFn: func(context.Context, *ps.CreateBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error) {
			return &ps.BillingPaymentMethodSetup{
				ID:          "pmsetup1",
				State:       "pending",
				CheckoutURL: "https://checkout.stripe.com/test",
			}, nil
		},
		GetFn: func(context.Context, *ps.GetBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error) {
			return &ps.BillingPaymentMethodSetup{
				ID:    "pmsetup1",
				State: "failed",
				Error: "The payment method could not be added.",
			}, nil
		},
	}

	var stdout bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&stdout)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethodSetups: service}, nil
		},
	}

	cmd := UpdatePaymentMethodCmd(ch)
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()

	cmdErr, ok := err.(*cmdutil.Error)
	if !ok || !cmdErr.Handled || cmdErr.ExitCode != cmdutil.FatalErrExitCode {
		t.Fatalf("error = %#v", err)
	}

	var problem paymentMethodProblem
	if err := json.Unmarshal(stdout.Bytes(), &problem); err != nil {
		t.Fatalf("stdout = %q: %v", stdout.String(), err)
	}
	if problem.Status != "error" || problem.ID != "pmsetup1" || problem.State != "failed" {
		t.Fatalf("problem = %#v", problem)
	}
	if len(problem.Issues) != 1 || problem.Issues[0].Code != "PAYMENT_METHOD_SETUP_FAILED" {
		t.Fatalf("issues = %#v", problem.Issues)
	}
	if problem.Error != "payment method setup pmsetup1 failed: The payment method could not be added." {
		t.Fatalf("error = %q", problem.Error)
	}
	// A failed setup cannot be resumed, so the only next step is a new setup.
	want := "pscale billing payment-method update --org my-org --format json"
	if len(problem.NextSteps) != 1 || problem.NextSteps[0] != want {
		t.Fatalf("next_steps = %#v", problem.NextSteps)
	}
}

func TestUpdatePaymentMethodJSONInterruptedEnvelope(t *testing.T) {
	oldInterval := paymentMethodSetupPollInterval
	oldOpenBrowser := openBrowser
	paymentMethodSetupPollInterval = time.Millisecond
	openBrowser = func(_ string, _ string) error { return errors.New("headless") }
	t.Cleanup(func() {
		paymentMethodSetupPollInterval = oldInterval
		openBrowser = oldOpenBrowser
	})

	ctx, cancel := context.WithCancel(context.Background())
	service := &mock.BillingPaymentMethodSetupsService{
		CreateFn: func(context.Context, *ps.CreateBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error) {
			return &ps.BillingPaymentMethodSetup{
				ID:          "pmsetup1",
				State:       "pending",
				CheckoutURL: "https://checkout.stripe.com/test",
			}, nil
		},
		GetFn: func(context.Context, *ps.GetBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error) {
			cancel()
			return &ps.BillingPaymentMethodSetup{ID: "pmsetup1", State: "pending"}, nil
		},
	}

	var stdout bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&stdout)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethodSetups: service}, nil
		},
	}

	cmd := UpdatePaymentMethodCmd(ch)
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.ExecuteContext(ctx)

	cmdErr, ok := err.(*cmdutil.Error)
	if !ok || cmdErr.ExitCode != cmdutil.ActionRequestedExitCode {
		t.Fatalf("error = %#v", err)
	}

	var problem paymentMethodProblem
	if err := json.Unmarshal(stdout.Bytes(), &problem); err != nil {
		t.Fatalf("stdout = %q: %v", stdout.String(), err)
	}
	if problem.Status != "action_required" || problem.State != "pending" {
		t.Fatalf("problem = %#v", problem)
	}
	if len(problem.Issues) != 1 || problem.Issues[0].Code != "PAYMENT_METHOD_SETUP_INTERRUPTED" {
		t.Fatalf("issues = %#v", problem.Issues)
	}
	// The Checkout session is still live, so resume rather than pay twice.
	want := "pscale billing payment-method status pmsetup1 --org my-org --format json"
	if len(problem.NextSteps) != 1 || problem.NextSteps[0] != want {
		t.Fatalf("next_steps = %#v", problem.NextSteps)
	}
}

func TestPaymentMethodStatus(t *testing.T) {
	service := &mock.BillingPaymentMethodSetupsService{
		GetFn: func(_ context.Context, req *ps.GetBillingPaymentMethodSetupRequest) (*ps.BillingPaymentMethodSetup, error) {
			if req.Organization != "my-org" || req.Setup != "pmsetup1" {
				t.Fatalf("request = %#v", req)
			}
			return &ps.BillingPaymentMethodSetup{ID: "pmsetup1", State: "pending", CheckoutURL: "https://checkout.stripe.com/test"}, nil
		},
	}

	var stdout bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&stdout)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethodSetups: service}, nil
		},
	}

	cmd := PaymentMethodStatusCmd(ch)
	cmd.SetArgs([]string{"pmsetup1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"checkout_url": "https://checkout.stripe.com/test"`)) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestShowPaymentMethod(t *testing.T) {
	service := &mock.BillingPaymentMethodsService{
		GetFn: func(_ context.Context, req *ps.GetBillingPaymentMethodRequest) (*ps.BillingPaymentMethod, error) {
			if req.Organization != "my-org" {
				t.Fatalf("organization = %q", req.Organization)
			}
			return &ps.BillingPaymentMethod{
				ID:       "pm_123",
				Brand:    "visa",
				Last4:    "4242",
				ExpMonth: 12,
				ExpYear:  2030,
			}, nil
		},
	}

	var stdout bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&stdout)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethods: service}, nil
		},
	}

	if err := ShowPaymentMethodCmd(ch).Execute(); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "{\n  \"id\": \"pm_123\",\n  \"brand\": \"visa\",\n  \"last4\": \"4242\",\n  \"exp_month\": 12,\n  \"exp_year\": 2030\n}\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestShowPaymentMethodNotFound(t *testing.T) {
	service := &mock.BillingPaymentMethodsService{
		GetFn: func(context.Context, *ps.GetBillingPaymentMethodRequest) (*ps.BillingPaymentMethod, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}
	format := printer.Human
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethods: service}, nil
		},
	}

	err := ShowPaymentMethodCmd(ch).Execute()
	if err == nil || err.Error() != "payment method does not exist in organization my-org" {
		t.Fatalf("error = %v", err)
	}
}

func TestShowPaymentMethodNotFoundJSON(t *testing.T) {
	service := &mock.BillingPaymentMethodsService{
		GetFn: func(context.Context, *ps.GetBillingPaymentMethodRequest) (*ps.BillingPaymentMethod, error) {
			return nil, &ps.Error{Code: ps.ErrNotFound}
		},
	}

	var stdout bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&stdout)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethods: service}, nil
		},
	}

	err := ShowPaymentMethodCmd(ch).Execute()
	cmdErr, ok := err.(*cmdutil.Error)
	if !ok || !cmdErr.Handled || cmdErr.ExitCode != cmdutil.FatalErrExitCode {
		t.Fatalf("error = %#v", err)
	}

	var problem paymentMethodProblem
	if err := json.Unmarshal(stdout.Bytes(), &problem); err != nil {
		t.Fatalf("stdout = %q: %v", stdout.String(), err)
	}
	if problem.Status != "error" || problem.Error != "payment method does not exist in organization my-org" {
		t.Fatalf("problem = %#v", problem)
	}
	if len(problem.Issues) != 1 || problem.Issues[0].Code != "NOT_FOUND" {
		t.Fatalf("issues = %#v", problem.Issues)
	}
	want := "pscale billing payment-method update --org my-org --format json"
	if len(problem.NextSteps) != 1 || problem.NextSteps[0] != want {
		t.Fatalf("next_steps = %#v", problem.NextSteps)
	}
}

func TestDeletePaymentMethod(t *testing.T) {
	service := &mock.BillingPaymentMethodsService{
		DeleteFn: func(_ context.Context, req *ps.DeleteBillingPaymentMethodRequest) error {
			if req.Organization != "my-org" {
				t.Fatalf("organization = %q", req.Organization)
			}
			return nil
		},
	}

	var stdout bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&stdout)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethods: service}, nil
		},
	}

	cmd := DeletePaymentMethodCmd(ch)
	cmd.SetArgs([]string{"--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !service.DeleteFnInvoked {
		t.Fatal("Delete was not called")
	}
	if got := stdout.String(); got != "{\n  \"organization\": \"my-org\",\n  \"result\": \"payment method deleted\"\n}\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestDeletePaymentMethodRequiresForceInJSON(t *testing.T) {
	service := &mock.BillingPaymentMethodsService{}
	format := printer.JSON
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethods: service}, nil
		},
	}

	err := DeletePaymentMethodCmd(ch).Execute()
	if err == nil || err.Error() == "" {
		t.Fatalf("error = %v", err)
	}
	if service.DeleteFnInvoked {
		t.Fatal("Delete should not be called without --force")
	}
}

func TestDeletePaymentMethodNotFoundJSON(t *testing.T) {
	service := &mock.BillingPaymentMethodsService{
		DeleteFn: func(context.Context, *ps.DeleteBillingPaymentMethodRequest) error {
			return &ps.Error{Code: ps.ErrNotFound}
		},
	}

	var stdout bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&stdout)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethods: service}, nil
		},
	}

	cmd := DeletePaymentMethodCmd(ch)
	cmd.SetArgs([]string{"--force"})
	err := cmd.Execute()
	cmdErr, ok := err.(*cmdutil.Error)
	if !ok || !cmdErr.Handled {
		t.Fatalf("error = %#v", err)
	}

	var problem paymentMethodProblem
	if err := json.Unmarshal(stdout.Bytes(), &problem); err != nil {
		t.Fatalf("stdout = %q: %v", stdout.String(), err)
	}
	if problem.Status != "error" || len(problem.Issues) != 1 || problem.Issues[0].Code != "NOT_FOUND" {
		t.Fatalf("problem = %#v", problem)
	}
	want := "pscale billing payment-method update --org my-org --format json"
	if len(problem.NextSteps) != 1 || problem.NextSteps[0] != want {
		t.Fatalf("next_steps = %#v", problem.NextSteps)
	}
}

func TestDeletePaymentMethodUnpaidInvoicesJSON(t *testing.T) {
	service := &mock.BillingPaymentMethodsService{
		DeleteFn: func(context.Context, *ps.DeleteBillingPaymentMethodRequest) error {
			return errors.New("Account has unpaid invoices. Please pay them before removing your card.")
		},
	}

	var stdout bytes.Buffer
	format := printer.JSON
	p := printer.NewPrinter(&format)
	p.SetResourceOutput(&stdout)
	ch := &cmdutil.Helper{
		Printer: p,
		Config:  &config.Config{Organization: "my-org"},
		Client: func() (*ps.Client, error) {
			return &ps.Client{PaymentMethods: service}, nil
		},
	}

	cmd := DeletePaymentMethodCmd(ch)
	cmd.SetArgs([]string{"--force"})
	err := cmd.Execute()
	cmdErr, ok := err.(*cmdutil.Error)
	if !ok || !cmdErr.Handled {
		t.Fatalf("error = %#v", err)
	}

	var problem paymentMethodProblem
	if err := json.Unmarshal(stdout.Bytes(), &problem); err != nil {
		t.Fatalf("stdout = %q: %v", stdout.String(), err)
	}
	if problem.Status != "error" || len(problem.Issues) != 1 || problem.Issues[0].Code != "UNPAID_INVOICES" {
		t.Fatalf("problem = %#v", problem)
	}
	if len(problem.NextSteps) != 1 || problem.NextSteps[0] != "Pay outstanding invoices, then retry this command" {
		t.Fatalf("next_steps = %#v", problem.NextSteps)
	}
}
