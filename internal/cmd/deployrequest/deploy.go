package deployrequest

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// DeployCmd is the command for deploying deploy requests.
func DeployCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		wait        bool
		instant_ddl bool
	}

	cmd := &cobra.Command{
		Use:   "deploy <database> <number|branch>",
		Short: "Deploy a specific deploy request",
		Args:  cmdutil.RequiredArgs("database", "number|branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			number_or_branch := args[1]
			var number uint64

			client, err := ch.Client()
			if err != nil {
				return err
			}

			number, err = strconv.ParseUint(number_or_branch, 10, 64)

			// Not a valid number, try branch name
			if err != nil {
				number, err = cmdutil.DeployRequestBranchToNumber(ctx, client, ch.Config.Organization, database, number_or_branch, "open")
				if err != nil {
					return err
				}
			}

			if flags.instant_ddl {
				ch.Printer.Printf("Deploy request %s/%s will be deployed instantly.\n\n",
					printer.BoldBlue(database), printer.BoldBlue(number))
			}

			dr, err := client.DeployRequests.Deploy(ctx, &planetscale.PerformDeployRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Number:       number,
				InstantDDL:   flags.instant_ddl,
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

			// wait and check until the deploy request is deployed
			if flags.wait {
				end := ch.Printer.PrintProgress(fmt.Sprintf("Waiting until deploy request %s/%s is deployed...",
					printer.BoldBlue(database), printer.BoldBlue(number)))
				defer end()
				getReq := &planetscale.GetDeployRequestRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Number:       number,
				}
				state, err := waitUntilReady(ctx, client, ch.Printer, ch.Debug(), getReq)
				if err != nil {
					return err
				}
				end()

				switch state {
				case "complete_pending_revert":
					ch.Printer.Printf("Deploy request %s/%s is successfully deployed and revertable. You can skip the revert to unblock the deploy queue.\n\n",
						printer.BoldBlue(database), printer.BoldBlue(number))
				case "pending_cutover":
					ch.Printer.Printf("Deploy request %s/%s is successfully staged and waiting to be applied.\n\n",
						printer.BoldBlue(database), printer.BoldBlue(number))
				default:
					ch.Printer.Printf("Deploy request %s/%s is successfully deployed.\n\n",
						printer.BoldBlue(database), printer.BoldBlue(number))
				}

			} else {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Successfully queued deploy request %s/%s from %s for deployment to %s.\n",
						printer.BoldBlue(database), printer.BoldBlue(number), printer.BoldBlue(dr.Branch), printer.BoldBlue(dr.IntoBranch))
					return nil
				}
			}

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	cmd.Flags().BoolVar(&flags.wait, "wait", false, "wait until the branch is deployed")
	cmd.Flags().BoolVar(&flags.instant_ddl, "instant", false, "If enabled, the schema migrations from this deploy request will be applied using MySQL’s built-in ALGORITHM=INSTANT option. Deployment will be faster, but cannot be reverted.")
	// cmd.Flags().MarkHidden("instant")

	return cmd
}

// waitUntilReady waits until the given deploy request has been deployed. It times out after 5 minutes.
func waitUntilReady(ctx context.Context, client *planetscale.Client, printer *printer.Printer, debug bool, getReq *planetscale.GetDeployRequestRequest) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return "", errors.New("deploy request queueing timed out")
		case <-ticker.C:
			resp, err := client.DeployRequests.Get(ctx, getReq)
			if err != nil {
				if debug {
					printer.Printf("fetching deploy request %s/%d failed: %s", getReq.Database, getReq.Number, err)
				}
				continue
			}

			if resp.Deployment.State == "complete" || resp.Deployment.State == "complete_pending_revert" || resp.Deployment.State == "pending_cutover" {
				return resp.Deployment.State, nil
			}
		}
	}
}
