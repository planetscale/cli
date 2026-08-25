package org

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		billingEmail     string
		idpManagedRoles  bool
		spendAlerts      bool
		spendAlertAmount int64
	}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an organization's settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &ps.UpdateOrganizationRequest{
				Organization: ch.Config.Organization,
			}

			changed := false
			if cmd.Flags().Changed("billing-email") {
				req.BillingEmail = &flags.billingEmail
				changed = true
			}
			if cmd.Flags().Changed("idp-managed-roles") {
				req.IDPManagedRoles = &flags.idpManagedRoles
				changed = true
			}

			if cmd.Flags().Changed("spend-alerts") || cmd.Flags().Changed("spend-alert-amount") {
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one of --billing-email, --idp-managed-roles, --spend-alerts, or --spend-alert-amount must be provided")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("spend-alerts") || cmd.Flags().Changed("spend-alert-amount") {
				if err := applySpendAlert(cmd, client, req, flags.spendAlerts, flags.spendAlertAmount); err != nil {
					return err
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating organization %s...", printer.BoldBlue(ch.Config.Organization)))
			defer end()

			updated, err := client.Organizations.Update(cmd.Context(), req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("organization %s does not exist", printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toOrganizationUpdate(updated))
		},
	}

	cmd.Flags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkFlagRequired("org")
	cmd.Flags().StringVar(&flags.billingEmail, "billing-email", "", "The billing email for the organization")
	cmd.Flags().BoolVar(&flags.idpManagedRoles, "idp-managed-roles", false, "Whether the identity provider manages organization roles")
	cmd.Flags().BoolVar(&flags.spendAlerts, "spend-alerts", false, "Enable or disable billing spend alerts")
	cmd.Flags().Int64Var(&flags.spendAlertAmount, "spend-alert-amount", 0, "Monthly spend amount that triggers spend alerts")

	return cmd
}

func applySpendAlert(cmd *cobra.Command, client *ps.Client, req *ps.UpdateOrganizationRequest, enabled bool, amount int64) error {
	alertChanged := cmd.Flags().Changed("spend-alerts")
	amountChanged := cmd.Flags().Changed("spend-alert-amount")

	if alertChanged && !enabled {
		req.InvoiceBudgetAlerts = boolPtr(false)
		req.ClearInvoiceBudget = true
		return nil
	}

	if amountChanged {
		req.InvoiceBudgetAmount = &amount
		req.InvoiceBudgetAlerts = boolPtr(true)
		return nil
	}

	org, err := client.Organizations.Get(cmd.Context(), &ps.GetOrganizationRequest{
		Organization: req.Organization,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case ps.ErrNotFound:
			return fmt.Errorf("organization %s does not exist", printer.BoldBlue(req.Organization))
		default:
			return cmdutil.HandleError(err)
		}
	}

	resolved, err := spendAlertAmount(org)
	if err != nil {
		return err
	}
	req.InvoiceBudgetAlerts = boolPtr(true)
	req.InvoiceBudgetAmount = &resolved
	return nil
}

func boolPtr(v bool) *bool { return &v }

func spendAlertAmount(org *ps.Organization) (int64, error) {
	for _, raw := range []ps.InvoiceBudgetAmount{org.InvoiceBudgetAmount, org.SuggestedInvoiceBudgetAmount} {
		n, err := strconv.ParseFloat(string(raw), 64)
		if err == nil && n > 0 {
			return int64(math.Round(n)), nil
		}
	}
	return 0, fmt.Errorf("no spend alert amount is set; pass --spend-alert-amount")
}

type organizationUpdate struct {
	Name             string `header:"name" json:"name"`
	BillingEmail     string `header:"billing_email" json:"billing_email"`
	IDPManagedRoles  bool   `header:"idp_managed_roles" json:"idp_managed_roles"`
	SpendAlerts      bool   `header:"spend_alerts" json:"spend_alerts"`
	SpendAlertAmount string `header:"spend_alert_amount" json:"spend_alert_amount"`

	orig *ps.Organization
}

func toOrganizationUpdate(org *ps.Organization) *organizationUpdate {
	return &organizationUpdate{
		Name:             org.Name,
		BillingEmail:     org.BillingEmail,
		IDPManagedRoles:  org.IDPManagedRoles,
		SpendAlerts:      org.InvoiceBudgetAlerts,
		SpendAlertAmount: string(org.InvoiceBudgetAmount),
		orig:             org,
	}
}

func (o *organizationUpdate) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(o.orig, "", "  ")
}

func (o *organizationUpdate) MarshalCSVValue() interface{} {
	return []*organizationUpdate{o}
}
