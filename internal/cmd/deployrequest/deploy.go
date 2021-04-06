package deployrequest

import (
	"context"
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// DeployCmd is the command for deploying deploy requests.
func DeployCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy <database> <number>",
		Short: "Deploy a specific deploy request by its number",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database := args[0]
			number := args[1]

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			n, err := strconv.ParseUint(number, 10, 64)
			if err != nil {
				return fmt.Errorf("The argument <number> is invalid: %s", err)
			}

			dr, err := client.DeployRequests.Deploy(ctx, &planetscale.PerformDeployRequest{
				Organization: cfg.Organization,
				Database:     database,
				Number:       n,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("%s/%s does not exist in %s\n",
						cmdutil.BoldBlue(database), cmdutil.BoldBlue(number), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			if cfg.OutputJSON {
				err := printer.PrintJSON(dr)
				if err != nil {
					return err
				}
			} else {
				fmt.Printf("Successfully deployed %s from %s to %s!\n", dr.ID, dr.Branch, dr.IntoBranch)
			}

			return nil
		},
	}

	return cmd
}
