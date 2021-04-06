package deployrequest

import (
	"context"
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// CloseCmd is the command for closing deploy requests.
func CloseCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close <database> <number>",
		Short: "Close deploy requests",
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

			_, err = client.DeployRequests.CloseDeploy(ctx, &planetscale.CloseDeployRequestRequest{
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

			fmt.Printf("Deploy request %s/%s was successfully closed!\n",
				cmdutil.BoldBlue(database), cmdutil.BoldBlue(number))
			return nil
		},
	}

	return cmd
}
