package oauthapplication

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func TokenDeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "delete <application-id> <token-id>",
		Short: "Delete an OAuth application token",
		Args:  cmdutil.RequiredArgs("application-id", "token-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			applicationID, tokenID := args[0], args[1]
			if !force {
				if err := ch.Printer.ConfirmCommand(tokenID, "delete oauth token", "deletion of oauth token"); err != nil {
					return err
				}
			}
			client, err := ch.Client()
			if err != nil {
				return err
			}
			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting OAuth token %s", printer.BoldBlue(tokenID)))
			defer end()
			err = client.OAuthApplications.DeleteToken(cmd.Context(), &ps.DeleteOAuthTokenRequest{
				Organization: ch.Config.Organization, ApplicationID: applicationID, TokenID: tokenID,
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
			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("OAuth token %s was successfully deleted.\n", printer.BoldBlue(tokenID))
				return nil
			}
			return ch.Printer.PrintResource(map[string]string{"result": "oauth token deleted"})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Delete the OAuth token without confirmation")
	return cmd
}
