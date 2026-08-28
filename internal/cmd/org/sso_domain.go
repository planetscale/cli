package org

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func SSODomainCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain <command>",
		Short: "List, verify, and delete SSO email domains",
	}

	cmd.AddCommand(SSODomainListCmd(ch))
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
				printHumanNextSteps(ch, []string{jsonSSOCmd(org, "domain verify")})
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationDomains(domains))
		},
	}

	return cmd
}

func SSODomainVerifyCmd(ch *cmdutil.Helper) *cobra.Command {
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating a domain verification URL for %s...", printer.BoldBlue(org)))
			defer end()

			portal, err := client.OrganizationSSO.VerifyDomain(ctx, &ps.OrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			return printSSOPortal(ch, portal.PortalURL, "verify", "verify an email domain")
		},
	}

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

			err = client.OrganizationSSO.DeleteDomain(ctx, &ps.DeleteOrganizationSSODomainRequest{
				Organization: org,
				DomainID:     domainID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("SSO domain %s does not exist in organization %s",
						printer.BoldBlue(domainID), printer.BoldBlue(org))
				default:
					return cmdutil.HandleError(err)
				}
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
