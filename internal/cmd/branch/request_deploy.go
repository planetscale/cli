package branch

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func RequestDeployCmd(cfg *config.Config) *cobra.Command {
	deployReq := &planetscale.DatabaseBranchRequestDeployRequest{}

	cmd := &cobra.Command{
		Use:   "request-deploy <database> <branch>",
		Short: "Requests a deploy for a specific schema snapshot ID",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database, branch := args[0], args[1]

			deployReq.Database = database
			deployReq.Branch = branch
			deployReq.Organization = cfg.Organization

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Request deploying of %s branch in %s...", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(database)))
			defer end()

			deployRequest, err := client.DatabaseBranches.RequestDeploy(ctx, deployReq)
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
				fmt.Printf("Successfully requested deploy %s of %s into %s!\n",
					cmdutil.BoldBlue(deployRequest.ID),
					cmdutil.BoldBlue(deployRequest.Branch),
					cmdutil.BoldBlue(deployRequest.IntoBranch))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&deployReq.Notes, "notes", "", "notes for the database")
	cmd.Flags().StringVar(&deployReq.IntoBranch, "into", "", "branch to deploy this schema snapshot into")

	return cmd
}
