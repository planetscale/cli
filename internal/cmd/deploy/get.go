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

func GetCmd(cfg *config.Config) *cobra.Command {
	getReq := &planetscale.GetDeployRequestRequest{}

	cmd := &cobra.Command{
		Use:   "get [id]",
		Short: "Get a deploy request",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) != 1 {
				return cmd.Usage()
			}

			id := args[0]
			getReq.ID = id

			end := cmdutil.PrintProgress(fmt.Sprintf("Fetching deploy %s...", cmdutil.BoldBlue(id)))
			defer end()

			dr, err := client.DeployRequests.Get(ctx, getReq)
			if err != nil {
				return err
			}
			end()

			isJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}

			err = printer.PrintOutput(isJSON, printer.NewDeployRequestPrinter(dr))
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
