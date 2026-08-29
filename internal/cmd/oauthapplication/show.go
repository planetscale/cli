package oauthapplication

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <application-id>",
		Short: "Show an OAuth application",
		Args:  cmdutil.RequiredArgs("application-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			applicationID := args[0]
			client, err := ch.Client()
			if err != nil {
				return err
			}
			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching OAuth application %s", printer.BoldBlue(applicationID)))
			defer end()
			application, err := client.OAuthApplications.Get(cmd.Context(), &ps.GetOAuthApplicationRequest{
				Organization: ch.Config.Organization,
				ID:           applicationID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("OAuth application %s does not exist in organization %s",
						printer.BoldBlue(applicationID), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()
			return ch.Printer.PrintResource(toOAuthApplication(application))
		},
	}
	return cmd
}
