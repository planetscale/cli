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
	var flags struct {
		filepath string
	}
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
					switch cmdutil.ErrCode(err) {
					case planetscale.ErrNotFound:
						return fmt.Errorf("%s does not exist\n", cmdutil.BoldBlue(orgName))
					case planetscale.ErrResponseMalformed:
						return cmdutil.MalformedError(err)
					default:
						return err
					}
				}
				end()
				organization = org.Name
			} else if len(args) == 0 && cmdutil.IsTTY {
				// Get organization names to show the user
				end := cmdutil.PrintProgress("Fetching organizations...")
				defer end()
				orgs, err := client.Organizations.List(ctx)
				if err != nil {
					switch cmdutil.ErrCode(err) {
					case planetscale.ErrResponseMalformed:
						return cmdutil.MalformedError(err)
					default:
						return err
					}
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
			fileCfg, err := config.NewFileConfig(filePath)
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

			fmt.Printf("Successfully switched to organization %s (using file: %s)\n",
				cmdutil.Bold(organization), filePath,
			)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.filepath, "save-config", "",
		"Path to store the organization. By default the configuration is deducted automatically based on where pscale is executed.")

	return cmd
}
