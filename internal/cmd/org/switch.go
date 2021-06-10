package org

import (
	"context"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
)

func SwitchCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		filepath string
	}
	cmd := &cobra.Command{
		Use:   "switch <organization>",
		Short: "Switch the currently active organization",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			client, err := ch.Client()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			orgs, err := client.Organizations.List(context.Background())
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			orgNames := make([]string, 0, len(orgs))
			for _, org := range orgs {
				orgNames = append(orgNames, org.Name)
			}

			return orgNames, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			organization := ""

			client, err := ch.Client()
			if err != nil {
				return err
			}

			// If user provides an organization, check if they have access to it.
			if len(args) == 1 {
				orgName := args[0]

				end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching organization %s...", printer.Bold(orgName)))
				defer end()
				org, err := client.Organizations.Get(ctx, &planetscale.GetOrganizationRequest{
					Organization: orgName,
				})
				if err != nil {
					switch cmdutil.ErrCode(err) {
					case planetscale.ErrNotFound:
						return fmt.Errorf("organization %s does not exist", printer.BoldBlue(orgName))
					default:
						return cmdutil.HandleError(err)
					}
				}
				end()
				organization = org.Name
			} else if len(args) == 0 && printer.IsTTY {
				// Get organization names to show the user
				end := ch.Printer.PrintProgress("Fetching organizations...")
				defer end()
				orgs, err := client.Organizations.List(ctx)
				if err != nil {
					return cmdutil.HandleError(err)
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

			filePath, _ := config.ProjectConfigPath()
			if _, err := os.Stat(filePath); err != nil {
				// clear the filePath if the file doesn't exist. We only switch
				// if the user explicilty is using a project specific file
				// configuration.
				filePath = ""
			}

			// if the user defined it explicitly, use it
			if flags.filepath != "" {
				filePath = flags.filepath
			}

			// fallback to the default global configuration path if nothing is
			// set.
			if filePath == "" {
				filePath, err = config.DefaultConfigPath()
				if err != nil {
					return err
				}
			}

			// check if a file already exists, we don't want to accidently
			// overwrite other values of the file config
			fileCfg, err := ch.ConfigFS.NewFileConfig(filePath)
			if os.IsNotExist(err) {
				// create a new file
				fileCfg = &config.FileConfig{
					Organization: organization,
				}
			} else {
				fileCfg.Organization = organization
				// TODO(fatih): check whether the branch/database exists for
				// the given organization and warn the user. The
				// branch/database combination will NOT be empty for a project
				// configuratin residing inside a Git repository.
			}

			err = fileCfg.Write(filePath)
			if err != nil {
				return err
			}

			ch.Printer.Printf("Successfully switched to organization %s (using file: %s)\n",
				printer.Bold(organization), filePath,
			)

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.filepath, "save-config", "",
		"Path to store the organization. By default the configuration is deducted automatically based on where pscale is executed.")

	return cmd
}
