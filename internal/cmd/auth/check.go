package auth

import (
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
			errorMessage := "You are not authenticated. Please run `pscale auth login` to authenticate."

			if ch.Config.AccessToken == "" {
				return &cmdutil.Error{
					Msg:      errorMessage,
					ExitCode: cmdutil.ActionRequestedExitCode,
				}
			} else {
				ctx := cmd.Context()
				client, err := ch.Client()
				if err != nil {
					return err
				}

				_, err = client.Organizations.List(ctx)
				if err != nil {
					return &cmdutil.Error{
						Msg:      errorMessage,
						ExitCode: cmdutil.ActionRequestedExitCode,
					}
				} else {
					ch.Printer.Println("You are authenticated.")
					return nil
				}
			}
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", auth.OAuthClientID, "The client ID for the PlanetScale CLI application.")
	cmd.Flags().StringVar(&clientSecret, "client-secret", auth.OAuthClientSecret, "The client ID for the PlanetScale CLI application")
	cmd.Flags().StringVar(&apiURL, "api-url", auth.DefaultBaseURL, "The PlanetScale base API URL.")
	return cmd
}
