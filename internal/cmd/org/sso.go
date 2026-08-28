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
		Short: "Manage organization SSO",
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

	nextSteps []string
	orig      *ps.OrganizationSSO
}

func (s *organizationSSO) MarshalJSON() ([]byte, error) {
	type payload struct {
		*ps.OrganizationSSO
		NextSteps []string `json:"next_steps"`
	}
	return json.MarshalIndent(payload{OrganizationSSO: s.orig, NextSteps: s.nextSteps}, "", "  ")
}

func (s *organizationSSO) MarshalCSVValue() interface{} {
	return []*organizationSSO{s}
}

func toOrganizationSSO(org string, sso *ps.OrganizationSSO) *organizationSSO {
	return &organizationSSO{
		ID:                sso.ID,
		Enabled:           sso.Enabled,
		Configured:        sso.Configured,
		Directory:         sso.Directory,
		HasVerifiedDomain: sso.HasVerifiedDomain,
		nextSteps:         ssoResourceNextSteps(org, sso),
		orig:              sso,
	}
}

type ssoPortal struct {
	PortalURL     string   `header:"portal_url" json:"portal_url"`
	BrowserOpened bool     `header:"browser_opened" json:"browser_opened"`
	NextSteps     []string `json:"next_steps"`
}

func (p *ssoPortal) MarshalCSVValue() interface{} {
	return []*ssoPortal{p}
}

type ssoDomainDeleted struct {
	Result    string   `json:"result"`
	Org       string   `json:"org"`
	ID        string   `json:"id"`
	NextSteps []string `json:"next_steps"`
}

func jsonSSOCmd(org, rest string) string {
	return fmt.Sprintf("pscale org sso %s --org %s --format json", rest, org)
}

func jsonOrgUpdateCmd(org, flag string) string {
	return fmt.Sprintf("pscale org update --org %s --format json %s", org, flag)
}

func ssoResourceNextSteps(org string, sso *ps.OrganizationSSO) []string {
	if sso == nil || !sso.Enabled {
		return []string{jsonSSOCmd(org, "enable")}
	}
	if !sso.HasVerifiedDomain {
		return []string{jsonSSOCmd(org, "domain verify")}
	}
	if !sso.Configured {
		return []string{jsonSSOCmd(org, "configure")}
	}
	if !sso.Directory {
		return []string{
			jsonSSOCmd(org, "directory enable"),
			jsonOrgUpdateCmd(org, "--idp-sso-managed-roles=true"),
		}
	}
	return []string{jsonOrgUpdateCmd(org, "--idp-managed-roles=true")}
}

func ssoPortalNextSteps(org, kind string, browserOpened bool) []string {
	steps := make([]string, 0, 3)
	if !browserOpened {
		steps = append(steps, "Open portal_url in a browser")
	}
	steps = append(steps, jsonSSOCmd(org, "show"))
	switch kind {
	case "configure":
		steps = append(steps, jsonOrgUpdateCmd(org, "--idp-sso-managed-roles=true"))
	case "directory":
		steps = append(steps, jsonOrgUpdateCmd(org, "--idp-managed-roles=true"))
	case "verify":
		steps = append(steps, jsonSSOCmd(org, "configure"))
	}
	return steps
}

func printHumanNextSteps(ch *cmdutil.Helper, steps []string) {
	if ch.Printer.Format() != printer.Human || len(steps) == 0 {
		return
	}
	ch.Printer.Println("Next:")
	for _, step := range steps {
		ch.Printer.Printf("  %s\n", step)
	}
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

func printSSOPortal(ch *cmdutil.Helper, url, kind, action string) error {
	if url == "" {
		return fmt.Errorf("no %s URL was returned", action)
	}

	org := ch.Config.Organization
	browserOpened := openSSOBrowser(runtime.GOOS, url) == nil
	steps := ssoPortalNextSteps(org, kind, browserOpened)
	if ch.Printer.Format() == printer.Human {
		if browserOpened {
			ch.Printer.Printf("Opened the %s URL in your browser.\n", action)
			ch.Printer.Printf("%s\n", printer.BoldBlue(url))
		} else {
			ch.Printer.Printf("Open this URL to %s: %s\n", action, printer.BoldBlue(url))
		}
		printHumanNextSteps(ch, steps)
		return nil
	}

	return ch.Printer.PrintResource(&ssoPortal{
		PortalURL:     url,
		BrowserOpened: browserOpened,
		NextSteps:     steps,
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

			sso, err := client.OrganizationSSO.Get(ctx, &ps.OrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			return ch.Printer.PrintResource(toOrganizationSSO(org, sso))
		},
	}

	return cmd
}

func SSOEnableCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable organization SSO",
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

			sso, err := client.OrganizationSSO.Enable(ctx, &ps.OrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("SSO is enabled for %s.\n", printer.BoldBlue(org))
				if sso.DomainVerificationURL != nil && *sso.DomainVerificationURL != "" {
					return printSSOPortal(ch, *sso.DomainVerificationURL, "verify", "verify an email domain")
				}
				printHumanNextSteps(ch, ssoResourceNextSteps(org, sso))
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationSSO(org, sso))
		},
	}

	return cmd
}

func SSODisableCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable organization SSO",
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

			sso, err := client.OrganizationSSO.Disable(ctx, &ps.OrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("SSO is disabled for %s.\n", printer.BoldBlue(org))
				printHumanNextSteps(ch, ssoResourceNextSteps(org, sso))
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationSSO(org, sso))
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

			portal, err := client.OrganizationSSO.Configure(ctx, &ps.OrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			return printSSOPortal(ch, portal.PortalURL, "configure", "configure SSO")
		},
	}

	return cmd
}
