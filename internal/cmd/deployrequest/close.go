package deployrequest

import (
	"context"
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// CloseCmd is the command for closing deploy requests.
func CloseCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close <database> <number>",
		Short: "Close a deploy request",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database := args[0]
			number := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			n, err := strconv.ParseUint(number, 10, 64)
			if err != nil {
				return fmt.Errorf("the argument <number> is invalid: %s", err)
			}

			dr, err := client.DeployRequests.CloseDeploy(ctx, &planetscale.CloseDeployRequestRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Number:       n,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("deploy request '%s/%s' does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(number), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Deploy request %s/%s was successfully closed.\n",
					printer.BoldBlue(database),
					printer.BoldBlue(dr.Number))
				return nil
			}

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	return cmd
}
