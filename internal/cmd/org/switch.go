package org

import (
	"context"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"

	"github.com/planetscale/planetscale-go/planetscale"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
)

func SwitchCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch <organization>",
		Short: "Switch the currently active organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			organization := ""

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			// If user provides an organization, check if they have access to it.
			if len(args) == 1 {
				orgName := args[0]

				end := cmdutil.PrintProgress(fmt.Sprintf("Fetching organization %s...", cmdutil.Bold(orgName)))
				defer end()
				org, err := client.Organizations.Get(ctx, &planetscale.GetOrganizationRequest{
					Organization: orgName,
				})
				if err != nil {
					return err
				}
				end()
				organization = org.Name
			} else if len(args) == 0 && cmdutil.IsTTY {
				// Get organization names to show the user
				end := cmdutil.PrintProgress("Fetching organizations...")
				defer end()
				orgs, err := client.Organizations.List(ctx)
				if err != nil {
					return err
				}
				end()

				orgNames := make([]string, 0, len(orgs))

				for _, org := range orgs {
					orgNames = append(orgNames, org.Name)
				}

				prompt := &survey.Select{
					Message: "Switch to: ",
					Options: orgNames,
				}

				err = survey.AskOne(prompt, &organization)
				if err != nil {
					if err == terminal.InterruptErr {
						os.Exit(0)
					} else {
						return err
					}
				}
			} else {
				return cmd.Usage()
			}

			globalCfg := &config.GlobalConfig{
				Organization: organization,
			}

			err = globalCfg.WriteDefault()
			if err != nil {
				return err
			}

			fmt.Printf("Successfully switched to organization %s\n", cmdutil.Bold(organization))
			return nil
		},
	}

	return cmd
}
