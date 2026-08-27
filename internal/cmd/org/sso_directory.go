package org

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func SSODirectoryCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "directory <command>",
		Short: "Enable or disable directory sync",
	}

	cmd.AddCommand(SSODirectoryEnableCmd(ch))
	cmd.AddCommand(SSODirectoryDisableCmd(ch))
	return cmd
}

func SSODirectoryEnableCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Open the directory sync setup portal",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating a directory sync URL for %s...", printer.BoldBlue(org)))
			defer end()

			portal, err := client.OrganizationSSO.EnableDirectory(ctx, &ps.OrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			return printSSOPortal(ch, portal.PortalURL, "configure directory sync")
		},
	}

	return cmd
}

func SSODirectoryDisableCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable directory sync",
		Long:  "Disable directory sync for the organization. Non-admin directory members are removed.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			if !force {
				if err := ch.Printer.ConfirmCommand(org, "disable directory sync", "disabling directory sync"); err != nil {
					return err
				}
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Disabling directory sync for %s...", printer.BoldBlue(org)))
			defer end()

			sso, err := client.OrganizationSSO.DisableDirectory(ctx, &ps.OrganizationSSORequest{Organization: org})
			if err != nil {
				return handleSSOError(org, err)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Directory sync is disabled for %s.\n", printer.BoldBlue(org))
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationSSO(sso))
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Disable directory sync without confirmation")
	return cmd
}
