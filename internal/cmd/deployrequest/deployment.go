package deployrequest

import (
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// DeploymentCmd shows the deployment details for a deploy request.
func DeploymentCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployment <database> <number>",
		Short: "Show the deployment for a deploy request",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			numberStr := args[1]

			number, err := strconv.ParseUint(numberStr, 10, 64)
			if err != nil {
				return fmt.Errorf("the argument <number> is invalid: %s", err)
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching deployment for %s/%s",
				printer.BoldBlue(database), printer.BoldBlue(number)))
			defer end()

			deployment, err := client.DeployRequests.GetDeployment(ctx, &planetscale.GetDeploymentRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Number:       number,
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
			end()

			return ch.Printer.PrintResource(toDeployment(deployment))
		},
	}

	return cmd
}
