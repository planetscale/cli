package token

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var name string
	var ttl string
	var cidrs []string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "update a service token's name, expiry, or IP restrictions",
		Long:  "Update a service token's name, expiry, or IP restrictions. This does not rotate the token secret or change its id or grants.",
		Example: `  pscale service-token update <token id> --name ci --ttl 3600
  pscale service-token update <token id> --cidrs 10.0.0.0/8,192.168.1.1/32
  pscale service-token update <token id> --cidrs ""`,
		Args: cmdutil.RequiredArgs("id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := ch.Client()
			if err != nil {
				return err
			}

			id := args[0]
			nameSet := cmd.Flags().Changed("name")
			ttlSet := cmd.Flags().Changed("ttl")
			cidrsSet := cmd.Flags().Changed("cidrs")
			if !nameSet && !ttlSet && !cidrsSet {
				return fmt.Errorf("at least one of --name, --ttl, or --cidrs must be provided")
			}

			req := &planetscale.UpdateServiceTokenRequest{
				Organization: ch.Config.Organization,
				ID:           id,
			}
			if nameSet {
				req.Name = &name
			}
			if ttlSet {
				parsed, err := serviceTokenTTL(ttl)
				if err != nil {
					return err
				}
				req.TTL = parsed
			}
			if cidrsSet {
				normalized := normalizeCIDRs(cidrs)
				req.CIDRs = &normalized
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating service token %s in org %s",
				printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization)))
			defer end()

			token, err := client.ServiceTokens.Update(ctx, req)
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

			if err := ch.Printer.PrintResource(toServiceTokenDetails(token)); err != nil {
				return err
			}
			if ch.Printer.Format() == printer.Human && cidrsSet {
				if len(token.CIDRs) == 0 {
					ch.Printer.Println("IP restrictions: none")
				} else {
					ch.Printer.Printf("IP restrictions: %s\n", strings.Join(token.CIDRs, ", "))
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "new name for the service token. Omit to leave the name unchanged")
	cmd.Flags().StringVar(&ttl, "ttl", "", `time to live in seconds, or "never" to remove the expiry. Omit to leave the expiry unchanged`)
	cmd.Flags().StringSliceVar(&cidrs, "cidrs", nil,
		"Replace the IP addresses and CIDR ranges allowed to use this service token. Pass an empty value to remove all restrictions")

	return cmd
}

// serviceTokenTTL parses the same seconds unit as service-token create.
// "never" clears the expiry. Create uses 0 for that and omits the field;
// update cannot, because an omitted --ttl leaves the expiry alone.
func serviceTokenTTL(v string) (*planetscale.UpdateServiceTokenTTL, error) {
	if v == "never" {
		return &planetscale.UpdateServiceTokenTTL{Never: true}, nil
	}
	seconds, err := strconv.Atoi(v)
	if err != nil {
		return nil, fmt.Errorf(`--ttl must be a number of seconds or "never"`)
	}
	return &planetscale.UpdateServiceTokenTTL{Seconds: seconds}, nil
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
