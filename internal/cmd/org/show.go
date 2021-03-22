package org

import (
	"context"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ShowCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the currently active organization",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			organization := ""

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if cmdutil.IsTTY {
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

			writableConfig := &config.WritableGlobalConfig{
				Organization: organization,
			}

			err = writableConfig.Write(viper.ConfigFileUsed())
			if err != nil {
				return err
			}

			fmt.Printf("Successfully switched to organization %s\n", cmdutil.Bold(organization))

			return nil
		},
	}

	return cmd
}
