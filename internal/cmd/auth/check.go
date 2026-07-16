package auth

import (
	psauth "github.com/planetscale/cli/internal/auth"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

func CheckCmd(ch *cmdutil.Helper) *cobra.Command {
	var clientID string
	var clientSecret string

	cmd := &cobra.Command{
		Use:   "check",
		Args:  cobra.NoArgs,
		Short: "Check if you are authenticated",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			resp := buildAuthCheckResponse(ctx, ch)

			if ch.Printer.Format() == printer.JSON {
				if err := ch.Printer.PrintJSON(resp); err != nil {
					return err
				}
				return authCheckExitCode(resp)
			}

			if !resp.Authenticated {
				msg := "You are not authenticated."
				if len(resp.Issues) > 0 {
					msg = resp.Issues[0].Message
				}
				return &cmdutil.Error{
					Msg:      msg,
					ExitCode: cmdutil.ActionRequestedExitCode,
				}
			}

			ch.Printer.Println("You are authenticated.")
			if resp.Organization != "" {
				ch.Printer.Printf("Organization: %s\n", resp.Organization)
			}
			for _, issue := range resp.Issues {
				ch.Printer.Printf("Note: %s — %s\n", issue.Message, issue.Remediation)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", psauth.OAuthClientID, "The client ID for the PlanetScale CLI application.")
	cmd.Flags().StringVar(&clientSecret, "client-secret", psauth.OAuthClientSecret, "The client ID for the PlanetScale CLI application")
	return cmd
}
