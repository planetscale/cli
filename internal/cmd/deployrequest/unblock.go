package deployrequest

import (
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// UnblockCmd unblocks the deploy queue after a failed deploy or revert.
func UnblockCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unblock <database> <number>",
		Short: "Unblock the deploy queue after a failed deploy or revert",
		Long: `Unblock the deploy queue after a failed deploy or revert.

When a deployment or revert errors, PlanetScale blocks the queue as a
precaution. This is the same action as "Unblock deploy queue" in the dashboard.
It does not apply a gated deploy (use 'deploy-request apply' for that) and it
does not fix a schema that failed deploy checks.

The API decides whether the failure was a deploy or a revert.`,
		Args: cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
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

			dr, err := client.DeployRequests.Get(ctx, &planetscale.GetDeployRequestRequest{
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

			if err := checkUnblockState(database, number, dr); err != nil {
				return err
			}

			dr, err = client.DeployRequests.UnblockDeploy(ctx, &planetscale.UnblockDeployRequestRequest{
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
				ch.Printer.Printf("Unblocked the deploy queue for '%s/%s'.\n",
					printer.BoldBlue(database),
					printer.BoldBlue(dr.Number))
				return nil
			}

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	return cmd
}

func checkUnblockState(database, number string, dr *planetscale.DeployRequest) error {
	if dr.Deployment == nil {
		return fmt.Errorf("deploy request '%s/%s' does not have a failed deploy to unblock",
			printer.BoldBlue(database), printer.BoldBlue(number))
	}

	switch dr.Deployment.State {
	case "complete_error", "complete_revert_error":
		return nil
	case "pending_cutover":
		return fmt.Errorf("deploy request '%s/%s' is waiting to apply changes; use 'pscale deploy-request apply %s %s'",
			printer.BoldBlue(database), printer.BoldBlue(number), database, number)
	case "error":
		msg := fmt.Sprintf("deploy request '%s/%s' failed deploy checks and cannot unblock the queue",
			printer.BoldBlue(database), printer.BoldBlue(number))
		if dr.Deployment.DeployCheckErrors != "" {
			msg += ": " + dr.Deployment.DeployCheckErrors
		}
		return fmt.Errorf("%s", msg)
	default:
		return fmt.Errorf("deploy request '%s/%s' does not have a failed deploy to unblock (deployment state: %s)",
			printer.BoldBlue(database), printer.BoldBlue(number), printer.BoldBlue(dr.Deployment.State))
	}
}
