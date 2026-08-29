package oauthapplication

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func TokenShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <application-id> <token-id>",
		Short: "Show an OAuth application token",
		Args:  cmdutil.RequiredArgs("application-id", "token-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			applicationID, tokenID := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}
			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching OAuth token %s", printer.BoldBlue(tokenID)))
			defer end()
			token, err := client.OAuthApplications.GetToken(cmd.Context(), &ps.GetOAuthTokenRequest{
				Organization:  ch.Config.Organization,
				ApplicationID: applicationID,
				TokenID:       tokenID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("OAuth token %s does not exist for application %s",
						printer.BoldBlue(tokenID), printer.BoldBlue(applicationID))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()
			return ch.Printer.PrintResource(toOAuthToken(token))
		},
	}
	return cmd
}
