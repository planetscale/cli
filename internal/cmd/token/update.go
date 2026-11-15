package token

import (
	"errors"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// UpdateCmd replaces the IP restrictions on a service token.
func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var cidrs []string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a service token's IP restrictions",
		Long: `Update the IP addresses and CIDR ranges allowed to use a service token.

This does not rotate the token. Pass an empty --cidrs value to remove all restrictions.`,
		Example: `  pscale service-token update <token id> --cidrs 10.0.0.0/8,192.168.1.1/32
  pscale service-token update <token id> --cidrs ""`,
		Args: cmdutil.RequiredArgs("id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			id := args[0]

			if !cmd.Flags().Changed("cidrs") {
				return errors.New("--cidrs must be provided")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating service token %s in org %s",
				printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization)))
			defer end()

			normalized := normalizeCIDRs(cidrs)
			token, err := client.ServiceTokens.Update(ctx, &planetscale.UpdateServiceTokenRequest{
				Organization: ch.Config.Organization,
				ID:           id,
				CIDRs:        &normalized,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("service token %s does not exist in organization %s",
						printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Service token %s was successfully updated.\n", printer.BoldBlue(token.ID))
				if len(token.CIDRs) == 0 {
					ch.Printer.Println("IP restrictions: none")
				} else {
					ch.Printer.Printf("IP restrictions: %s\n", strings.Join(token.CIDRs, ", "))
				}
				return nil
			}

			return ch.Printer.PrintResource(toServiceTokenDetails(token))
		},
	}

	cmd.Flags().StringSliceVar(&cidrs, "cidrs", nil,
		"Replace the IP addresses and CIDR ranges allowed to use this service token. Pass an empty value to remove all restrictions")

	return cmd
}

// normalizeCIDRs drops the empty entries produced by flags like --cidrs "" so
// that clearing restrictions sends an empty list instead of a blank CIDR.
func normalizeCIDRs(cidrs []string) []string {
	out := make([]string, 0, len(cidrs))
	for _, cidr := range cidrs {
		if trimmed := strings.TrimSpace(cidr); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
