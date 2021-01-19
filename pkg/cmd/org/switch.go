package org

import (
	"context"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

func SwitchCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch [organization]",
		Short: "Switch the currently active organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			answers := &struct {
				Organization string
			}{}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}
			// If user provides an organization, check if they have access to it.
			if len(args) == 1 {
				orgName := args[0]

				org, err := client.Organizations.Get(ctx, orgName)
				if err != nil {
					return err
				}

				answers.Organization = org.Name
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

				qs := []*survey.Question{
					{
						Name: "organization",
						Prompt: &survey.Select{
							Message: "Switch to: ",
							Options: orgNames,
							Default: cfg.Organization,
						},
					},
				}

				err = survey.Ask(qs, answers)
				if err != nil {
					return err
				}
			} else {
				return cmd.Usage()
			}

			// TODO: Write config for `answers.Organization` here.
			bold := color.New(color.Bold).SprintFunc()
			fmt.Printf("Successfully switched to organization %s.\n", bold(answers.Organization))

			return nil
		},
	}

	return cmd
}
