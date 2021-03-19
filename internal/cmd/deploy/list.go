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

func ListCmd(cfg *config.Config) *cobra.Command {
	listReq := &planetscale.ListDeployRequestsRequest{}

	cmd := &cobra.Command{
		Use:   "list <database> <branch>",
		Short: "List the deploy requests for a database branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database, branch := args[0], args[1]

			listReq.Organization = cfg.Organization
			listReq.Database = database
			listReq.Branch = branch

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Fetching deploys for %s in %s...", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(database)))
			defer end()

			deployRequests, err := client.DatabaseBranches.ListDeployRequests(ctx, listReq)
			if err != nil {
				return err
			}
			end()

			isJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}

			err = printer.PrintOutput(isJSON, printer.NewDeployRequestSlicePrinter(deployRequests))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
