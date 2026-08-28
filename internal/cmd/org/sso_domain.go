package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

var ssoDomainPollInterval = 2 * time.Second

func SSODomainCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain <command>",
		Short: "List, show, verify, and delete SSO email domains",
	}

	cmd.AddCommand(SSODomainListCmd(ch))
	cmd.AddCommand(SSODomainShowCmd(ch))
	cmd.AddCommand(SSODomainVerifyCmd(ch))
	cmd.AddCommand(SSODomainDeleteCmd(ch))
	return cmd
}

func SSODomainListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List SSO email domains",
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching SSO domains for %s...", printer.BoldBlue(org)))
			defer end()

			domains, err := client.OrganizationSSO.ListDomains(ctx, &ps.OrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			if len(domains) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No SSO domains in %s.\n", printer.BoldBlue(org))
				printHumanNextSteps(ch, []string{jsonSSOCmd(org, "domain verify") + " --wait"})
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationDomains(org, domains))
		},
	}

	return cmd
}

func SSODomainShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <domain-id>",
		Short: "Show an SSO email domain",
		Args:  cmdutil.RequiredArgs("domain-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization
			domainID := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching SSO domain %s...", printer.BoldBlue(domainID)))
			defer end()

			domain, err := client.OrganizationSSO.GetDomain(ctx, &ps.OrganizationSSODomainRequest{
				Organization: org,
				DomainID:     domainID,
			})
			if err != nil {
				return handleSSODomainError(org, domainID, err)
			}
			end()

			return printSSODomain(ch, org, domain)
		},
	}

	return cmd
}

func SSODomainVerifyCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		wait        bool
		waitTimeout time.Duration
	}

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Open the domain verification portal",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			var existing []*ps.OrganizationDomain
			if flags.wait {
				existing, err = client.OrganizationSSO.ListDomains(ctx, &ps.OrganizationSSORequest{Organization: org})
				if err != nil {
					return handleSSOError(org, err)
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating a domain verification URL for %s...", printer.BoldBlue(org)))
			defer func() { end() }()

			portal, err := client.OrganizationSSO.VerifyDomain(ctx, &ps.OrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			if !flags.wait {
				return printSSOPortal(ch, portal.PortalURL, "verify", "verify an email domain")
			}

			if err := printSSOPortalPending(cmd, ch, portal.PortalURL, "verify an email domain"); err != nil {
				return err
			}

			end = ch.Printer.PrintProgress(fmt.Sprintf("Waiting for an SSO domain to verify in %s...", printer.BoldBlue(org)))
			waitCtx, cancel := context.WithTimeout(ctx, flags.waitTimeout)
			defer cancel()
			domain, err := waitForSSODomainVerification(waitCtx, client, org, existing)
			if err != nil {
				return err
			}
			end()

			return printSSODomain(ch, org, domain)
		},
	}

	cmd.Flags().BoolVar(&flags.wait, "wait", false, "After opening the portal, wait until a domain is verified")
	cmd.Flags().DurationVar(&flags.waitTimeout, "wait-timeout", 10*time.Minute, "Maximum time to wait with --wait")
	return cmd
}

func SSODomainDeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <domain-id>",
		Short:   "Delete an SSO email domain",
		Aliases: []string{"rm"},
		Args:    cmdutil.RequiredArgs("domain-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization
			domainID := args[0]

			if !force {
				if err := ch.Printer.ConfirmCommand(domainID, "delete SSO domain", "deletion of SSO domain"); err != nil {
					return err
				}
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting SSO domain %s from %s...", printer.BoldBlue(domainID), printer.BoldBlue(org)))
			defer end()

			err = client.OrganizationSSO.DeleteDomain(ctx, &ps.OrganizationSSODomainRequest{
				Organization: org,
				DomainID:     domainID,
			})
			if err != nil {
				return handleSSODomainError(org, domainID, err)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Deleted SSO domain %s from %s.\n",
					printer.BoldBlue(domainID), printer.BoldBlue(org))
				printHumanNextSteps(ch, []string{jsonSSOCmd(org, "domain list"), jsonSSOCmd(org, "show")})
				return nil
			}

			return ch.Printer.PrintResource(&ssoDomainDeleted{
				Result:    "domain deleted",
				Org:       org,
				ID:        domainID,
				NextSteps: []string{jsonSSOCmd(org, "domain list"), jsonSSOCmd(org, "show")},
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete the domain without confirmation")
	return cmd
}

func printSSODomain(ch *cmdutil.Helper, org string, domain *ps.OrganizationDomain) error {
	return ch.Printer.PrintResource(toOrganizationDomain(org, domain))
}

func printSSOPortalPending(cmd *cobra.Command, ch *cmdutil.Helper, url, action string) error {
	if url == "" {
		return fmt.Errorf("no %s URL was returned", action)
	}

	browserOpened := openSSOBrowser(runtime.GOOS, url) == nil
	if ch.Printer.Format() == printer.JSON {
		pending := &ssoPortal{
			PortalURL:     url,
			BrowserOpened: browserOpened,
			NextSteps:     ssoPortalNextSteps(ch.Config.Organization, "verify", browserOpened),
		}
		return json.NewEncoder(cmd.ErrOrStderr()).Encode(pending)
	}

	if browserOpened {
		ch.Printer.Printf("Opened the %s URL in your browser.\n", action)
		ch.Printer.Printf("%s\n", printer.BoldBlue(url))
	} else {
		ch.Printer.Printf("Open this URL to %s: %s\n", action, printer.BoldBlue(url))
	}
	return nil
}

func waitUntilSSODomainReady(ctx context.Context, client *ps.Client, org, domainID string) (*ps.OrganizationDomain, error) {
	ticker := time.NewTicker(ssoDomainPollInterval)
	defer ticker.Stop()

	for {
		domain, err := client.OrganizationSSO.GetDomain(ctx, &ps.OrganizationSSODomainRequest{
			Organization: org,
			DomainID:     domainID,
		})
		if err != nil {
			if timedOut(err, ctx) {
				return nil, fmt.Errorf("timed out waiting for SSO domain %s", domainID)
			}
			if !retryableSSOPollError(err) {
				return nil, handleSSODomainError(org, domainID, err)
			}
		} else {
			switch domain.State {
			case "verified":
				return domain, nil
			case "failed":
				return domain, ssoDomainFailedError(domain)
			}
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for SSO domain %s", domainID)
		case <-ticker.C:
		}
	}
}

func waitForSSODomainVerification(ctx context.Context, client *ps.Client, org string, existing []*ps.OrganizationDomain) (*ps.OrganizationDomain, error) {
	ticker := time.NewTicker(ssoDomainPollInterval)
	defer ticker.Stop()

	known := make(map[string]struct{}, len(existing))
	for _, domain := range existing {
		known[domain.ID] = struct{}{}
	}

	current := existing
	for {
		if domain := latestNewDomain(known, current); domain != nil {
			return waitUntilSSODomainReady(ctx, client, org, domain.ID)
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for an SSO domain in %s", org)
		case <-ticker.C:
		}

		domains, err := client.OrganizationSSO.ListDomains(ctx, &ps.OrganizationSSORequest{Organization: org})
		if err != nil {
			if timedOut(err, ctx) {
				return nil, fmt.Errorf("timed out waiting for an SSO domain in %s", org)
			}
			if !retryableSSOPollError(err) {
				return nil, handleSSOError(org, err)
			}
			continue
		}
		current = domains
	}
}

func timedOut(err error, ctx context.Context) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil
}

func retryableSSOPollError(err error) bool {
	switch cmdutil.ErrCode(err) {
	case ps.ErrPermission, ps.ErrInvalid, ps.ErrResponseMalformed:
		return false
	default:
		return true
	}
}

func latestNewDomain(known map[string]struct{}, current []*ps.OrganizationDomain) *ps.OrganizationDomain {
	var latest *ps.OrganizationDomain
	for _, domain := range current {
		if _, ok := known[domain.ID]; ok {
			continue
		}
		if latest == nil || domain.CreatedAt.After(latest.CreatedAt) {
			latest = domain
		}
	}
	return latest
}

func ssoDomainFailedError(domain *ps.OrganizationDomain) error {
	if domain.FailureReason != nil && *domain.FailureReason != "" {
		return fmt.Errorf("SSO domain %s failed: %s", domain.ID, *domain.FailureReason)
	}
	return fmt.Errorf("SSO domain %s failed verification", domain.ID)
}

func handleSSODomainError(org, domainID string, err error) error {
	switch cmdutil.ErrCode(err) {
	case ps.ErrNotFound:
		return fmt.Errorf("SSO domain %s does not exist in organization %s",
			printer.BoldBlue(domainID), printer.BoldBlue(org))
	default:
		return cmdutil.HandleError(err)
	}
}
