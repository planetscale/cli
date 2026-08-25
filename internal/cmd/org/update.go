package org

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		billingEmail        string
		idpManagedRoles     bool
		invoiceBudgetAmount float64
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
			if cmd.Flags().Changed("invoice-budget-amount") {
				req.InvoiceBudgetAmount = &flags.invoiceBudgetAmount
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one of --billing-email, --idp-managed-roles, or --invoice-budget-amount must be provided")
			}

			client, err := ch.Client()
			if err != nil {
				return err
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
	cmd.Flags().Float64Var(&flags.invoiceBudgetAmount, "invoice-budget-amount", 0, "The expected monthly budget for the organization")

	return cmd
}

type organizationUpdate struct {
	Name                string  `header:"name" json:"name"`
	BillingEmail        string  `header:"billing_email" json:"billing_email"`
	IDPManagedRoles     bool    `header:"idp_managed_roles" json:"idp_managed_roles"`
	InvoiceBudgetAmount float64 `header:"invoice_budget_amount" json:"invoice_budget_amount"`

	orig *ps.Organization
}

func toOrganizationUpdate(org *ps.Organization) *organizationUpdate {
	return &organizationUpdate{
		Name:                org.Name,
		BillingEmail:        org.BillingEmail,
		IDPManagedRoles:     org.IDPManagedRoles,
		InvoiceBudgetAmount: org.InvoiceBudgetAmount,
		orig:                org,
	}
}

func (o *organizationUpdate) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(o.orig, "", "  ")
}

func (o *organizationUpdate) MarshalCSVValue() interface{} {
	return []*organizationUpdate{o}
}
