package oidctrust

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "show <trust-id>",
		Short: "Show an OIDC trust",
		Args:  cmdutil.RequiredArgs("trust-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching OIDC trust %s", printer.BoldBlue(id)))
			defer end()

			trust, err := client.OIDCTrusts.Get(cmd.Context(), &planetscale.GetOIDCTrustRequest{
				Organization: ch.Config.Organization,
				ID:           id,
			})
			if err != nil {
				if cmdutil.ErrCode(err) == planetscale.ErrNotFound {
					return fmt.Errorf("OIDC trust %s does not exist in organization %s",
						printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization))
				}
				return cmdutil.HandleError(err)
			}

			end()

			return ch.Printer.PrintResource(toOIDCTrust(trust))
		},
	}
}
