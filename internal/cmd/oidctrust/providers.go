package oidctrust

import (
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// Provider is a table and CSV serializable OIDC trust provider. JSON output
// uses the API objects directly so that required claims remain an array.
type Provider struct {
	Name           string `header:"name" json:"name"`
	DisplayName    string `header:"display name" json:"display_name"`
	Issuer         string `header:"issuer" json:"issuer"`
	RequiredClaims string `header:"required claims" json:"required_claims"`
}

// MarshalCSVValue wraps a single provider in a slice, which the CSV encoder
// requires.
func (p *Provider) MarshalCSVValue() interface{} {
	return []*Provider{p}
}

// ProviderClaim is a table serializable claim of one OIDC trust provider.
type ProviderClaim struct {
	Name        string `header:"claim" json:"name"`
	Required    bool   `header:"required" json:"required"`
	Description string `header:"description" json:"description"`
}

func toProvider(p *planetscale.OIDCTrustProvider) *Provider {
	return &Provider{
		Name:           p.Name,
		DisplayName:    p.DisplayName,
		Issuer:         p.Issuer,
		RequiredClaims: strings.Join(p.RequiredClaims, ", "),
	}
}

// ProvidersCmd encapsulates the commands for OIDC trust providers. Running it
// without a subcommand lists providers, as it did before it had subcommands.
func ProvidersCmd(ch *cmdutil.Helper) *cobra.Command {
	list := ProvidersListCmd(ch)

	cmd := &cobra.Command{
		Use:   "providers <command>",
		Short: "List and show OIDC trust providers",
		Args:  cobra.NoArgs,
		RunE:  list.RunE,
	}

	cmd.AddCommand(list)
	cmd.AddCommand(ProvidersShowCmd(ch))

	return cmd
}

func ProvidersListCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List OIDC trust providers",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			providers, err := listProviders(cmd, ch)
			if err != nil {
				return err
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(providers)
			}

			results := make([]*Provider, 0, len(providers))
			for _, p := range providers {
				results = append(results, toProvider(p))
			}
			return ch.Printer.PrintResource(results)
		},
	}
}

func ProvidersShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show an OIDC trust provider and the claims in its tokens",
		Args:  cmdutil.RequiredArgs("name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			providers, err := listProviders(cmd, ch)
			if err != nil {
				return err
			}

			var provider *planetscale.OIDCTrustProvider
			for _, p := range providers {
				if p.Name == name {
					provider = p
					break
				}
			}
			if provider == nil {
				return fmt.Errorf("OIDC trust provider %s is not available in organization %s",
					printer.BoldBlue(name), printer.BoldBlue(ch.Config.Organization))
			}

			if ch.Printer.Format() == printer.JSON {
				return ch.Printer.PrintJSON(provider)
			}

			if err := ch.Printer.PrintResource(toProvider(provider)); err != nil {
				return err
			}

			if ch.Printer.Format() != printer.Human || len(provider.Claims) == 0 {
				return nil
			}

			claims := make([]*ProviderClaim, 0, len(provider.Claims))
			for _, c := range provider.Claims {
				claims = append(claims, &ProviderClaim{Name: c.Name, Required: c.Required, Description: c.Description})
			}
			return ch.Printer.PrintResource(claims)
		},
	}
}

func listProviders(cmd *cobra.Command, ch *cmdutil.Helper) ([]*planetscale.OIDCTrustProvider, error) {
	client, err := ch.Client()
	if err != nil {
		return nil, err
	}

	end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching OIDC trust providers for %s", printer.BoldBlue(ch.Config.Organization)))
	defer end()

	providers, err := client.OIDCTrusts.ListProviders(cmd.Context(), &planetscale.ListOIDCTrustProvidersRequest{
		Organization: ch.Config.Organization,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case planetscale.ErrNotFound:
			return nil, fmt.Errorf("organization %s does not exist or does not have OIDC trusts enabled",
				printer.BoldBlue(ch.Config.Organization))
		default:
			return nil, cmdutil.HandleError(err)
		}
	}

	return providers, nil
}
