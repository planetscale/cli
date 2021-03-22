package deploy

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func DeployCmd(cfg *config.Config) *cobra.Command {
	performReq := &planetscale.PerformDeployRequest{}

	cmd := &cobra.Command{
		Use:   "deploy <id>",
		Short: "Approve and deploy a specific deploy request",
		Args:  cmdutil.RequiredArgs("id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			id := args[0]

			performReq.ID = id

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Attempting to finish deploy for %s", cmdutil.BoldBlue(id)))
			defer end()

			deployRequest, err := client.DeployRequests.Deploy(ctx, performReq)
			if err != nil {
				return err
			}
			end()

			if cfg.OutputJSON {
				err := printer.PrintJSON(deployRequest)
				if err != nil {
					return err
				}
			} else {
				fmt.Printf("Successfully deployed %s from %s to %s!\n", deployRequest.ID, deployRequest.Branch, deployRequest.IntoBranch)
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(cfg))
	cmd.AddCommand(GetCmd(cfg))

	return cmd
}
