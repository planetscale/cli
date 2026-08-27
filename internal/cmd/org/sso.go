package org

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

var openSSOBrowser = cmdutil.TryOpenBrowser

func SSOCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sso <command>",
		Short: "Enable, configure, and disable organization SSO",
		Long: `Manage organization single sign-on.

Enabling SSO returns a URL for verifying an email domain. Configuring SSO and
directory sync opens a setup URL. Requires an org admin session, or a token
with the manage_sso grant.`,
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org")

	cmd.AddCommand(SSOShowCmd(ch))
	cmd.AddCommand(SSOEnableCmd(ch))
	cmd.AddCommand(SSODisableCmd(ch))
	cmd.AddCommand(SSOConfigureCmd(ch))
	cmd.AddCommand(SSODirectoryCmd(ch))
	cmd.AddCommand(SSODomainCmd(ch))

	return cmd
}

type organizationSSO struct {
	ID                string `header:"id" json:"id"`
	Enabled           bool   `header:"enabled" json:"enabled"`
	Configured        bool   `header:"configured" json:"configured"`
	Directory         bool   `header:"directory" json:"directory"`
	HasVerifiedDomain bool   `header:"has_verified_domain" json:"has_verified_domain"`

	orig *ps.OrganizationSSO
}

func (s *organizationSSO) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(s.orig, "", "  ")
}

func (s *organizationSSO) MarshalCSVValue() interface{} {
	return []*organizationSSO{s}
}

func toOrganizationSSO(sso *ps.OrganizationSSO) *organizationSSO {
	return &organizationSSO{
		ID:                sso.ID,
		Enabled:           sso.Enabled,
		Configured:        sso.Configured,
		Directory:         sso.Directory,
		HasVerifiedDomain: sso.HasVerifiedDomain,
		orig:              sso,
	}
}

type ssoPortal struct {
	PortalURL     string `header:"portal_url" json:"portal_url"`
	BrowserOpened bool   `header:"browser_opened" json:"browser_opened"`
}

func (p *ssoPortal) MarshalCSVValue() interface{} {
	return []*ssoPortal{p}
}

type organizationDomain struct {
	ID     string `header:"id" json:"id"`
	Domain string `header:"domain" json:"domain"`
	State  string `header:"state" json:"state"`

	orig *ps.OrganizationDomain
}

func (d *organizationDomain) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *organizationDomain) MarshalCSVValue() interface{} {
	return []*organizationDomain{d}
}

func toOrganizationDomains(domains []*ps.OrganizationDomain) []*organizationDomain {
	out := make([]*organizationDomain, 0, len(domains))
	for _, domain := range domains {
		out = append(out, &organizationDomain{
			ID:     domain.ID,
			Domain: domain.Domain,
			State:  domain.State,
			orig:   domain,
		})
	}
	return out
}

func handleSSOError(org string, err error) error {
	switch cmdutil.ErrCode(err) {
	case ps.ErrNotFound:
		return fmt.Errorf("organization %s does not exist", printer.BoldBlue(org))
	default:
		return cmdutil.HandleError(err)
	}
}

func printSSOPortal(ch *cmdutil.Helper, url, action string) error {
	if url == "" {
		return fmt.Errorf("no %s URL was returned", action)
	}

	browserOpened := openSSOBrowser(runtime.GOOS, url) == nil
	if ch.Printer.Format() == printer.Human {
		if browserOpened {
			ch.Printer.Printf("Opened the %s URL in your browser.\n", action)
			ch.Printer.Printf("%s\n", printer.BoldBlue(url))
		} else {
			ch.Printer.Printf("Open this URL to %s: %s\n", action, printer.BoldBlue(url))
		}
		return nil
	}

	return ch.Printer.PrintResource(&ssoPortal{
		PortalURL:     url,
		BrowserOpened: browserOpened,
	})
}

func SSOShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show organization SSO status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching SSO status for %s...", printer.BoldBlue(org)))
			defer end()

			sso, err := client.OrganizationSSO.Get(ctx, &ps.GetOrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			return ch.Printer.PrintResource(toOrganizationSSO(sso))
		},
	}

	return cmd
}

func SSOEnableCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable organization SSO",
		Long:  "Enable the SSO add-on and return a URL for verifying an email domain.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Enabling SSO for %s...", printer.BoldBlue(org)))
			defer end()

			sso, err := client.OrganizationSSO.Enable(ctx, &ps.EnableOrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			if sso.DomainVerificationURL != nil && *sso.DomainVerificationURL != "" {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("SSO is enabled for %s.\n", printer.BoldBlue(org))
					return printSSOPortal(ch, *sso.DomainVerificationURL, "verify an email domain")
				}
				_ = openSSOBrowser(runtime.GOOS, *sso.DomainVerificationURL)
			} else if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("SSO is enabled for %s.\n", printer.BoldBlue(org))
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationSSO(sso))
		},
	}

	return cmd
}

func SSODisableCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable organization SSO",
		Long:  "Disable SSO and directory sync for the organization.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			if !force {
				if err := ch.Printer.ConfirmCommand(org, "disable SSO", "disabling SSO"); err != nil {
					return err
				}
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Disabling SSO for %s...", printer.BoldBlue(org)))
			defer end()

			sso, err := client.OrganizationSSO.Disable(ctx, &ps.DisableOrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("SSO is disabled for %s.\n", printer.BoldBlue(org))
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationSSO(sso))
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Disable SSO without confirmation")
	return cmd
}

func SSOConfigureCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Open the identity provider setup portal",
		Long:  "Return a URL for configuring the identity provider. Requires SSO to be enabled and at least one verified domain.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating an SSO configuration URL for %s...", printer.BoldBlue(org)))
			defer end()

			portal, err := client.OrganizationSSO.Configure(ctx, &ps.ConfigureOrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			return printSSOPortal(ch, portal.PortalURL, "configure SSO")
		},
	}

	return cmd
}
