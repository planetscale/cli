package auth

import (
	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	"github.com/planetscale/cli/internal/auth"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func RefreshCmd(ch *cmdutil.Helper) *cobra.Command {
	var clientID string
	var clientSecret string
	var authURL string

	cmd := &cobra.Command{
		Use:   "refresh",
		Args:  cobra.ExactArgs(0),
		Short: "Refresh authentication with the PlanetScale API",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !printer.IsTTY {
				return errors.New("The 'login' command requires an interactive shell")
			}

			authenticator, err := auth.New(cleanhttp.DefaultClient(), clientID, clientSecret, auth.SetBaseURL(authURL))
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			end := ch.Printer.PrintProgress("Refreshing authentication...")
			defer end()
			accessToken, refreshToken, err := authenticator.RefreshToken(ctx, ch.Config.RefreshToken)
			if err != nil {
				return err
			}

			err = writeAccessToken(ctx, accessToken)
			if err != nil {
				return errors.Wrap(err, "error refreshing token")
			}

			err = writeRefreshToken(ctx, refreshToken)
			if err != nil {
				return errors.Wrap(err, "error refreshing token")
			}

			// we explicitly stop here so we can replace the spinner with our success
			// message.
			end()
			ch.Printer.Println("Successfully refreshed authentication.")

			return nil
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", auth.OAuthClientID, "The client ID for the PlanetScale CLI application.")
	cmd.Flags().StringVar(&clientSecret, "client-secret", auth.OAuthClientSecret, "The client ID for the PlanetScale CLI application")
	cmd.Flags().StringVar(&authURL, "api-url", auth.DefaultBaseURL, "The PlanetScale Auth API base URL.")

	return cmd
}
