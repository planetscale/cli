package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

var (
	paymentMethodSetupPollInterval = 2 * time.Second
	openBrowser                    = cmdutil.TryOpenBrowser
)

type PaymentMethodSetup struct {
	ID          string     `header:"id" json:"id"`
	State       string     `header:"state" json:"state"`
	CheckoutURL string     `header:"checkout_url" json:"checkout_url,omitempty"`
	Error       string     `header:"error" json:"error,omitempty"`
	ExpiresAt   *time.Time `header:"expires_at" json:"expires_at,omitempty"`
	CompletedAt *time.Time `header:"completed_at" json:"completed_at,omitempty"`
	FailedAt    *time.Time `header:"failed_at" json:"failed_at,omitempty"`

	orig *ps.BillingPaymentMethodSetup
}

func (s *PaymentMethodSetup) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(s.orig, "", "  ")
}

func (s *PaymentMethodSetup) MarshalCSVValue() interface{} {
	return []*PaymentMethodSetup{s}
}

func toPaymentMethodSetup(setup *ps.BillingPaymentMethodSetup) *PaymentMethodSetup {
	return &PaymentMethodSetup{
		ID:          setup.ID,
		State:       setup.State,
		CheckoutURL: setup.CheckoutURL,
		Error:       setup.Error,
		ExpiresAt:   setup.ExpiresAt,
		CompletedAt: setup.CompletedAt,
		FailedAt:    setup.FailedAt,
		orig:        setup,
	}
}

type PaymentMethod struct {
	ID       string `header:"id" json:"id"`
	Brand    string `header:"brand" json:"brand"`
	Last4    string `header:"last4" json:"last4"`
	ExpMonth int    `header:"exp_month" json:"exp_month"`
	ExpYear  int    `header:"exp_year" json:"exp_year"`
	Name     string `header:"name" json:"name,omitempty"`

	orig *ps.BillingPaymentMethod
}

func (c *PaymentMethod) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(c.orig, "", "  ")
}

func (c *PaymentMethod) MarshalCSVValue() interface{} {
	return []*PaymentMethod{c}
}

func toPaymentMethod(card *ps.BillingPaymentMethod) *PaymentMethod {
	return &PaymentMethod{
		ID:       card.ID,
		Brand:    card.Brand,
		Last4:    card.Last4,
		ExpMonth: card.ExpMonth,
		ExpYear:  card.ExpYear,
		Name:     card.Name,
		orig:     card,
	}
}

type paymentMethodSetupPending struct {
	Status        string   `json:"status"`
	ID            string   `json:"id"`
	CheckoutURL   string   `json:"checkout_url"`
	BrowserOpened bool     `json:"browser_opened"`
	Message       string   `json:"message"`
	NextSteps     []string `json:"next_steps"`
}

// paymentMethodProblem is the standard JSON error envelope, plus optional
// setup identity so a caller can tell a terminal setup (start a new one)
// from an interrupted poll (resume with status).
type paymentMethodProblem struct {
	Status    string                   `json:"status"`
	Error     string                   `json:"error"`
	ID        string                   `json:"id,omitempty"`
	State     string                   `json:"state,omitempty"`
	Issues    []cmdutil.JSONErrorIssue `json:"issues"`
	NextSteps []string                 `json:"next_steps"`
}

func UpdatePaymentMethodCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the payment method using Stripe Checkout",
		Long:  "Create a Stripe-hosted Checkout link, open it in a browser when possible, and wait for the payment method to be verified and saved. The setup id printed here is what `pscale billing payment-method status` takes if this command is interrupted.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}

			setup, err := client.PaymentMethodSetups.Create(cmd.Context(), &ps.CreateBillingPaymentMethodSetupRequest{
				Organization: ch.Config.Organization,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			if setup.ID == "" || setup.CheckoutURL == "" {
				return errors.New("payment method setup response is missing an ID or Checkout URL")
			}

			browserOpened := openBrowser(runtime.GOOS, setup.CheckoutURL) == nil
			statusCommand := fmt.Sprintf("pscale billing payment-method status %s --org %s", setup.ID, ch.Config.Organization)
			if ch.Printer.Format() == printer.JSON {
				statusCommand += " --format json"
			}
			updateCommand := paymentMethodUpdateCommand(ch.Config.Organization)

			if ch.Printer.Format() == printer.JSON {
				pending := paymentMethodSetupPending{
					Status:        "pending",
					ID:            setup.ID,
					CheckoutURL:   setup.CheckoutURL,
					BrowserOpened: browserOpened,
					Message:       paymentMethodSetupMessage(browserOpened),
					NextSteps:     []string{"Complete Stripe Checkout", statusCommand},
				}
				if err := json.NewEncoder(cmd.ErrOrStderr()).Encode(pending); err != nil {
					return err
				}
			} else {
				ch.Printer.Printf("Payment method setup: %s\n", printer.BoldBlue(setup.ID))
				ch.Printer.Printf("If this is interrupted, check it with: %s\n", printer.BoldBlue(statusCommand))
				if browserOpened {
					ch.Printer.Printf("Complete Stripe Checkout in your browser: %s\n", printer.BoldBlue(setup.CheckoutURL))
				} else {
					ch.Printer.Printf("Open this URL to complete Stripe Checkout: %s\n", printer.BoldBlue(setup.CheckoutURL))
				}
			}

			end := func() {}
			if ch.Printer.Format() == printer.Human {
				end = ch.Printer.PrintProgress("Waiting for payment method verification...")
			}
			progressStopped := false
			defer func() {
				if !progressStopped {
					end()
				}
			}()

			polled, err := pollPaymentMethodSetup(cmd.Context(), client.PaymentMethodSetups, ch.Config.Organization, setup.ID)
			if err != nil {
				end()
				progressStopped = true
				// The setup is still live in Stripe, so the recovery is resuming
				// this ID rather than paying for a second Checkout session.
				return paymentMethodProblemError(
					ch,
					"action_required",
					"PAYMENT_METHOD_SETUP_INTERRUPTED",
					setup.ID,
					"pending",
					fmt.Sprintf("%s; resume with %s", err, statusCommand),
					[]string{statusCommand},
					cmdutil.ActionRequestedExitCode,
				)
			}
			setup = polled
			end()
			progressStopped = true

			switch setup.State {
			case "completed":
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Payment method updated for organization %s.\n", printer.BoldBlue(ch.Config.Organization))
					return nil
				}
				return ch.Printer.PrintResource(toPaymentMethodSetup(setup))
			default:
				// Every non-completed terminal state is unrecoverable: the
				// Checkout session is spent, so a new setup is the only way
				// forward.
				return paymentMethodProblemError(
					ch,
					"error",
					paymentMethodSetupFailureCode(setup.State),
					setup.ID,
					setup.State,
					paymentMethodSetupFailureMessage(setup),
					[]string{updateCommand},
					cmdutil.FatalErrExitCode,
				)
			}
		},
	}
	return cmd
}

func PaymentMethodStatusCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <setup-id>",
		Short: "Show a payment method setup started by update",
		Long:  "Show a Checkout setup. Pass the setup id printed by `pscale billing payment-method update`, not the saved card id from `show`.",
		Args:  cmdutil.RequiredArgs("setup-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}
			setup, err := client.PaymentMethodSetups.Get(cmd.Context(), &ps.GetBillingPaymentMethodSetupRequest{
				Organization: ch.Config.Organization,
				Setup:        args[0],
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			return printPaymentMethodSetup(ch, setup)
		},
	}
	return cmd
}

func ShowPaymentMethodCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the organization payment method",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}

			card, err := client.PaymentMethods.Get(cmd.Context(), &ps.GetBillingPaymentMethodRequest{
				Organization: ch.Config.Organization,
			})
			if err != nil {
				return handlePaymentMethodAPIError(ch, err)
			}

			return ch.Printer.PrintResource(toPaymentMethod(card))
		},
	}
	return cmd
}

func DeletePaymentMethodCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete the organization payment method",
		Aliases: []string{"rm"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				if err := ch.Printer.ConfirmCommand(ch.Config.Organization, "delete payment method", "deletion of payment method"); err != nil {
					return err
				}
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			err = client.PaymentMethods.Delete(cmd.Context(), &ps.DeleteBillingPaymentMethodRequest{
				Organization: ch.Config.Organization,
			})
			if err != nil {
				return handlePaymentMethodAPIError(ch, err)
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Payment method deleted from organization %s.\n", printer.BoldBlue(ch.Config.Organization))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result":       "payment method deleted",
				"organization": ch.Config.Organization,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete the payment method without confirmation")
	return cmd
}

func pollPaymentMethodSetup(ctx context.Context, service ps.BillingPaymentMethodSetupsService, organization, setupID string) (*ps.BillingPaymentMethodSetup, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("payment method setup %s interrupted: %w", setupID, ctx.Err())
		case <-time.After(paymentMethodSetupPollInterval):
		}

		setup, err := service.Get(ctx, &ps.GetBillingPaymentMethodSetupRequest{
			Organization: organization,
			Setup:        setupID,
		})
		if err != nil {
			return nil, cmdutil.HandleError(err)
		}
		if setup.State != "pending" {
			return setup, nil
		}
	}
}

func printPaymentMethodSetup(ch *cmdutil.Helper, setup *ps.BillingPaymentMethodSetup) error {
	if ch.Printer.Format() != printer.Human {
		return ch.Printer.PrintResource(toPaymentMethodSetup(setup))
	}

	ch.Printer.Printf("%-16s %s\n", "ID", setup.ID)
	ch.Printer.Printf("%-16s %s\n", "State", setup.State)
	if setup.CheckoutURL != "" {
		ch.Printer.Printf("%-16s %s\n", "Checkout URL", setup.CheckoutURL)
	}
	if setup.Error != "" {
		ch.Printer.Printf("%-16s %s\n", "Error", setup.Error)
	}
	return nil
}

func paymentMethodSetupMessage(browserOpened bool) string {
	if browserOpened {
		return "Complete Stripe Checkout in the browser to continue"
	}
	return "Open checkout_url in a browser to continue"
}

func paymentMethodSetupFailureCode(state string) string {
	if state == "expired" {
		return "PAYMENT_METHOD_SETUP_EXPIRED"
	}
	return "PAYMENT_METHOD_SETUP_FAILED"
}

func paymentMethodSetupFailureMessage(setup *ps.BillingPaymentMethodSetup) string {
	switch {
	case setup.State == "expired":
		return fmt.Sprintf("payment method setup %s expired before Checkout completed", setup.ID)
	case setup.State != "failed":
		return fmt.Sprintf("payment method setup %s reached unexpected state %q", setup.ID, setup.State)
	case setup.Error == "":
		return fmt.Sprintf("payment method setup %s failed", setup.ID)
	default:
		return fmt.Sprintf("payment method setup %s failed: %s", setup.ID, setup.Error)
	}
}

func paymentMethodUpdateCommand(org string) string {
	return fmt.Sprintf("pscale billing payment-method update --org %s --format json", org)
}

func handlePaymentMethodAPIError(ch *cmdutil.Helper, err error) error {
	err = cmdutil.HandleError(err)
	org := ch.Config.Organization
	humanMsg := fmt.Sprintf("payment method does not exist in organization %s", printer.BoldBlue(org))
	jsonMsg := fmt.Sprintf("payment method does not exist in organization %s", org)

	switch cmdutil.ErrCode(err) {
	case ps.ErrNotFound:
		if ch.Printer.Format() != printer.JSON {
			return errors.New(humanMsg)
		}
		return paymentMethodProblemError(
			ch,
			"error",
			"NOT_FOUND",
			"",
			"",
			jsonMsg,
			[]string{paymentMethodUpdateCommand(org)},
			cmdutil.FatalErrExitCode,
		)
	}

	if err != nil && strings.Contains(strings.ToLower(err.Error()), "unpaid invoices") {
		msg := err.Error()
		if ch.Printer.Format() != printer.JSON {
			return err
		}
		return paymentMethodProblemError(
			ch,
			"error",
			"UNPAID_INVOICES",
			"",
			"",
			msg,
			[]string{"Pay outstanding invoices, then retry this command"},
			cmdutil.FatalErrExitCode,
		)
	}

	return err
}

// paymentMethodProblemError prints the JSON envelope itself so command-specific
// codes and next_steps survive; human format falls back to a plain error.
func paymentMethodProblemError(ch *cmdutil.Helper, status, code, id, state, message string, nextSteps []string, exitCode int) error {
	if ch.Printer.Format() != printer.JSON {
		return errors.New(message)
	}

	problem := paymentMethodProblem{
		Status:    status,
		Error:     message,
		ID:        id,
		State:     state,
		Issues:    []cmdutil.JSONErrorIssue{{Code: code, Message: message}},
		NextSteps: nextSteps,
	}
	if err := ch.Printer.PrintJSON(problem); err != nil {
		return err
	}
	return cmdutil.JSONReportedError(exitCode)
}
