package org

import (
	"context"
	"fmt"
	"os"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/fatih/color"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func SwitchCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch [organization]",
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

				if orgName == cfg.Organization {
					color.White("No changes made")
					return nil
				}

				org, err := client.Organizations.Get(ctx, orgName)
				if err != nil {
					return err
				}

				organization = org.Name
			} else if len(args) == 0 {
				// Get organization names to show the user
				orgs, err := client.Organizations.List(ctx)
				if err != nil {
					return err
				}

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

			writableConfig := &config.WritableConfig{
				Organization: organization,
			}

			err = writableConfig.Write(viper.ConfigFileUsed())
			if err != nil {
				return err
			}

			bold := color.New(color.Bold).SprintFunc()
			fmt.Printf("Successfully switched to organization %s\n", bold(organization))

			return nil
		},
	}

	return cmd
}
