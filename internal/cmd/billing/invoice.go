package billing

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func InvoiceCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoice <command>",
		Short: "List and show organization invoices",
	}
	cmd.AddCommand(ListInvoicesCmd(ch))
	cmd.AddCommand(ShowInvoiceCmd(ch))
	cmd.AddCommand(InvoiceLineItemsCmd(ch))
	return cmd
}

type invoice struct {
	ID                 string `header:"id" json:"id"`
	Total              string `header:"total" json:"total"`
	BillingPeriodStart string `header:"billing_period_start" json:"billing_period_start"`
	BillingPeriodEnd   string `header:"billing_period_end" json:"billing_period_end"`
	Paid               bool   `header:"paid" json:"paid"`
	Overdue            bool   `header:"overdue" json:"overdue"`

	orig *ps.Invoice
}

func (i *invoice) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(i.orig, "", "  ")
}

func (i *invoice) MarshalCSVValue() interface{} {
	return []*invoice{i}
}

func toInvoices(invoices []*ps.Invoice) []*invoice {
	out := make([]*invoice, 0, len(invoices))
	for _, inv := range invoices {
		out = append(out, toInvoice(inv))
	}
	return out
}

func toInvoice(inv *ps.Invoice) *invoice {
	return &invoice{
		ID:                 inv.ID,
		Total:              inv.Total,
		BillingPeriodStart: inv.BillingPeriodStart,
		BillingPeriodEnd:   inv.BillingPeriodEnd,
		Paid:               inv.Paid,
		Overdue:            inv.Overdue,
		orig:               inv,
	}
}

type invoiceLineItem struct {
	ID               string `header:"id" json:"id"`
	Description      string `header:"description" json:"description"`
	MetricName       string `header:"metric_name" json:"metric_name"`
	Subtotal         string `header:"subtotal" json:"subtotal"`
	DatabaseName     string `header:"database" json:"database_name"`
	ResourceName     string `header:"resource" json:"resource_name"`
	CloudflareBilled bool   `header:"cloudflare_billed" json:"cloudflare_billed"`

	orig *ps.InvoiceLineItem
}

func (i *invoiceLineItem) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(i.orig, "", "  ")
}

func (i *invoiceLineItem) MarshalCSVValue() interface{} {
	return []*invoiceLineItem{i}
}

func toInvoiceLineItems(items []*ps.InvoiceLineItem) []*invoiceLineItem {
	out := make([]*invoiceLineItem, 0, len(items))
	for _, item := range items {
		out = append(out, toInvoiceLineItem(item))
	}
	return out
}

func toInvoiceLineItem(item *ps.InvoiceLineItem) *invoiceLineItem {
	return &invoiceLineItem{
		ID:               item.ID,
		Description:      item.Description,
		MetricName:       item.MetricName,
		Subtotal:         string(item.Subtotal),
		DatabaseName:     item.DatabaseName,
		ResourceName:     item.Resource.Name,
		CloudflareBilled: item.CloudflareBilled,
		orig:             item,
	}
}

func ListInvoicesCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List invoices for an organization",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			org := ch.Config.Organization
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching invoices for %s...", printer.BoldBlue(org)))
			defer end()

			page, err := client.Invoices.List(cmd.Context(), &ps.ListInvoicesRequest{
				Organization: org,
			}, ps.WithPage(flags.page), ps.WithPerPage(flags.perPage))
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("organization %s does not exist", printer.BoldBlue(org))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(page.Data) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Println("No invoices found on this page.")
				return nil
			}

			if err := ch.Printer.PrintResource(toInvoices(page.Data)); err != nil {
				return err
			}
			printNextPageHint(ch, page.NextPage)
			return nil
		},
	}

	cmd.Flags().IntVar(&flags.page, "page", 1, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 25, "Number of results per page")
	return cmd
}

func ShowInvoiceCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <invoice-id>",
		Short: "Show an invoice",
		Args:  cmdutil.RequiredArgs("invoice-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			org := ch.Config.Organization
			id := args[0]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching invoice %s in %s...", printer.BoldBlue(id), printer.BoldBlue(org)))
			defer end()

			invoice, err := client.Invoices.Get(cmd.Context(), &ps.GetInvoiceRequest{
				Organization: org,
				Invoice:      id,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("invoice %s does not exist in organization %s", printer.BoldBlue(id), printer.BoldBlue(org))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toInvoice(invoice))
		},
	}

	return cmd
}

func InvoiceLineItemsCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:   "line-items <invoice-id>",
		Short: "List line items for an invoice",
		Args:  cmdutil.RequiredArgs("invoice-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			org := ch.Config.Organization
			id := args[0]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching line items for invoice %s in %s...", printer.BoldBlue(id), printer.BoldBlue(org)))
			defer end()

			page, err := client.Invoices.ListLineItems(cmd.Context(), &ps.ListInvoiceLineItemsRequest{
				Organization: org,
				Invoice:      id,
			}, ps.WithPage(flags.page), ps.WithPerPage(flags.perPage))
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("invoice %s does not exist in organization %s", printer.BoldBlue(id), printer.BoldBlue(org))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(page.Data) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Println("No line items found on this page.")
				return nil
			}

			if err := ch.Printer.PrintResource(toInvoiceLineItems(page.Data)); err != nil {
				return err
			}
			printNextPageHint(ch, page.NextPage)
			return nil
		},
	}

	cmd.Flags().IntVar(&flags.page, "page", 1, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 25, "Number of results per page")
	return cmd
}

// printNextPageHint tells a human there is another page. JSON and CSV output
// stay machine-readable, so callers there compare the row count to --per-page.
func printNextPageHint(ch *cmdutil.Helper, nextPage *int) {
	if nextPage == nil || ch.Printer.Format() != printer.Human {
		return
	}
	ch.Printer.Printf("\nMore results available. Fetch the next page with --page %d.\n", *nextPage)
}
