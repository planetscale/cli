package billing

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func BillingCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "billing <command>",
		Short:             "Manage organization billing",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(PaymentMethodCmd(ch))
	cmd.AddCommand(InvoiceCmd(ch))
	return cmd
}

func PaymentMethodCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "payment-method <command>",
		Short: "Manage the organization payment method",
	}
	cmd.AddCommand(UpdatePaymentMethodCmd(ch))
	cmd.AddCommand(ShowPaymentMethodCmd(ch))
	cmd.AddCommand(DeletePaymentMethodCmd(ch))
	cmd.AddCommand(PaymentMethodStatusCmd(ch))
	return cmd
}
