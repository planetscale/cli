package oidctrust

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List OIDC trusts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching OIDC trusts for %s", printer.BoldBlue(ch.Config.Organization)))
			defer end()

			trusts, err := client.OIDCTrusts.List(cmd.Context(), &planetscale.ListOIDCTrustsRequest{
				Organization: ch.Config.Organization,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("organization %s does not exist or does not have OIDC trusts enabled",
						printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if len(trusts) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No OIDC trusts exist in organization %s.\n", printer.BoldBlue(ch.Config.Organization))
				return nil
			}

			out := make([]*OIDCTrust, 0, len(trusts))
			for _, t := range trusts {
				out = append(out, toOIDCTrust(t))
			}

			return ch.Printer.PrintResource(out)
		},
	}
}
