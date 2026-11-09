package oidctrust

import (
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// Provider returns a table and json serializable OIDC provider for printing.
type Provider struct {
	Name           string `header:"name" json:"name"`
	DisplayName    string `header:"display name" json:"display_name"`
	Issuer         string `header:"issuer" json:"issuer"`
	RequiredClaims string `header:"required claims" json:"required_claims"`
}

func ProvidersCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List OIDC providers available for trusts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching OIDC providers for %s", printer.BoldBlue(ch.Config.Organization)))
			defer end()

			providers, err := client.OIDCTrusts.ListProviders(cmd.Context(), &planetscale.ListOIDCTrustProvidersRequest{
				Organization: ch.Config.Organization,
			})
			if err != nil {
				if cmdutil.ErrCode(err) == planetscale.ErrNotFound {
					return orgNotEnabledError(ch.Config.Organization)
				}
				return cmdutil.HandleError(err)
			}

			end()

			results := make([]*Provider, 0, len(providers))
			for _, p := range providers {
				results = append(results, &Provider{
					Name:           p.Name,
					DisplayName:    p.DisplayName,
					Issuer:         p.Issuer,
					RequiredClaims: strings.Join(p.RequiredClaims, ", "),
				})
			}
			return ch.Printer.PrintResource(results)
		},
	}
}
