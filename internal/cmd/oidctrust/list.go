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
		Short: "List OIDC trusts for an organization",
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
				if cmdutil.ErrCode(err) == planetscale.ErrNotFound {
					return orgNotEnabledError(ch.Config.Organization)
				}
				return cmdutil.HandleError(err)
			}

			end()

			if len(trusts) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No OIDC trusts exist in organization %s.\n", printer.BoldBlue(ch.Config.Organization))
				return nil
			}

			return ch.Printer.PrintResource(toOIDCTrusts(trusts))
		},
	}
}
