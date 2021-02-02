package snapshot

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/cmdutil"
	"github.com/planetscale/cli/config"
	"github.com/planetscale/cli/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func RequestDeployCmd(cfg *config.Config) *cobra.Command {
	deployReq := &planetscale.SchemaSnapshotRequestDeployRequest{}

	cmd := &cobra.Command{
		Use:   "request-deploy <id>",
		Short: "Requests a deploy for a specific schema snapshot ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if len(args) != 1 {
				return cmd.Usage()
			}

			id := args[0]
			deployReq.SchemaSnapshotID = id

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Request deploying of schema snapshot %s...", cmdutil.BoldBlue(id)))
			defer end()
			deployRequest, err := client.SchemaSnapshots.RequestDeploy(ctx, deployReq)
			if err != nil {
				return err
			}
			end()

			isJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			if isJSON {
				err := printer.PrintJSON(deployRequest)
				if err != nil {
					return err
				}
			} else {
				fmt.Printf("Successfully requested deploy %s for %s into %s!\n",
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
