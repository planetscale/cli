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
		wait bool
	}

	cmd := &cobra.Command{
		Use:   "deploy <database> <number>",
		Short: "Deploy a specific deploy request",
		Args:  cmdutil.RequiredArgs("database", "number"),
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

			dr, err := client.DeployRequests.Deploy(ctx, &planetscale.PerformDeployRequest{
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

			// wait and check until the deploy request is deployed
			if flags.wait {
				end := ch.Printer.PrintProgress(fmt.Sprintf("Waiting until deploy request %s/%s is deployed...",
					printer.BoldBlue(database), printer.BoldBlue(number)))
				defer end()
				getReq := &planetscale.GetDeployRequestRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Number:       n,
				}
				if err := waitUntilReady(ctx, client, ch.Printer, ch.Debug(), getReq); err != nil {
					return err
				}
				end()

				ch.Printer.Printf("Deploy request %s/%s is successfully deployed.",
					printer.BoldBlue(database), printer.BoldBlue(number))

			} else {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Successfully queued %s from %s for deployment to %s.\n",
						dr.ID, dr.Branch, dr.IntoBranch)
					return nil
				}
			}

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	cmd.Flags().BoolVar(&flags.wait, "wait", false, "wait until the branch is deployed")

	return cmd
}

// waitUntilReady waits until the given deploy request has been deployed. It times out after 3 minutes.
func waitUntilReady(ctx context.Context, client *planetscale.Client, printer *printer.Printer, debug bool, getReq *planetscale.GetDeployRequestRequest) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return errors.New("deploy request queueing timed out")
		case <-ticker.C:
			resp, err := client.DeployRequests.Get(ctx, getReq)
			if err != nil {
				if debug {
					printer.Printf("fetching deploy request %s/%d failed: %s", getReq.Database, getReq.Number, err)
				}
				continue
			}

			if resp.State == "complete_pending_revert" {
				return nil
			}
		}
	}
}
