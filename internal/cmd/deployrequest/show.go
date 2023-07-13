package deployrequest

import (
	"fmt"
	"strconv"

	"github.com/pkg/browser"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// ShowCmd is the command to show a deploy request.
func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		web bool
	}

	cmd := &cobra.Command{
		Use:   "show <database> <number|branch>",
		Short: "Show a specific deploy request",
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
				number, err = cmdutil.DeployRequestBranchToNumber(ctx, client, ch.Config.Organization, database, number_or_branch, "")
				if err != nil {
					return err
				}
			}

			if flags.web {
				ch.Printer.Println("🌐  Redirecting you to your deploy request in your web browser.")
				return browser.OpenURL(fmt.Sprintf("%s/%s/%s/deploy-requests/%s", cmdutil.ApplicationURL, ch.Config.Organization, database, strconv.FormatUint(number, 10)))
			}

			dr, err := client.DeployRequests.Get(ctx, &planetscale.GetDeployRequestRequest{
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

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	cmd.PersistentFlags().BoolVar(&flags.web, "web", false, "Open in your web browser")

	return cmd
}
