package auth

import (
	"fmt"
	"github.com/planetscale/cli/internal/auth"
	"github.com/planetscale/cli/internal/cmdutil"

	"github.com/spf13/cobra"
)

func CheckCmd(ch *cmdutil.Helper) *cobra.Command {
	var clientID string
	var clientSecret string
	var apiURL string

	cmd := &cobra.Command{
		Use:   "check",
		Args:  cobra.NoArgs,
		Short: "Check if you are authenticated",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ch.Config.AccessToken == "" {
				return &cmdutil.Error{
					Msg:      fmt.Sprintf("Not authenticated. Use pscale auth login."),
					ExitCode: cmdutil.ActionRequestedExitCode,
				}
			} else {
				ch.Printer.Printf("You are authenticated.")
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", auth.OAuthClientID, "The client ID for the PlanetScale CLI application.")
	cmd.Flags().StringVar(&clientSecret, "client-secret", auth.OAuthClientSecret, "The client ID for the PlanetScale CLI application")
	cmd.Flags().StringVar(&apiURL, "api-url", auth.DefaultBaseURL, "The PlanetScale base API URL.")
	return cmd
}
