package auth

import (
	"bufio"
	"io"
	"os"

	"github.com/planetscale/cli/internal/auth"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func LogoutCmd(ch *cmdutil.Helper) *cobra.Command {
	var clientID string
	var clientSecret string
	var apiURL string

	cmd := &cobra.Command{
		Use:   "logout",
		Args:  cobra.NoArgs,
		Short: "Log out of the PlanetScale API",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ch.Config.AccessToken == "" {
				ch.Printer.Println("Already logged out. Exiting...")
				return nil
			}

			if printer.IsTTY {
				ch.Printer.Println("Press Enter to log out of the PlanetScale API.")
				_ = waitForEnter(cmd.InOrStdin())
			}

			authenticator, err := auth.New(cleanhttp.DefaultClient(), clientID, clientSecret, auth.SetBaseURL(apiURL))
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			end := ch.Printer.PrintProgress("Logging out...")
			defer end()
			err = authenticator.RevokeToken(ctx, ch.Config.AccessToken)
			if err != nil {
				return err
			}
			err = deleteAccessToken()
			if err != nil {
				return err
			}
			end()
			ch.Printer.Println("Successfully logged out.")

			return nil
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", auth.OAuthClientID, "The client ID for the PlanetScale CLI application.")
	cmd.Flags().StringVar(&clientSecret, "client-secret", auth.OAuthClientSecret, "The client ID for the PlanetScale CLI application")
	cmd.Flags().StringVar(&apiURL, "api-url", auth.DefaultBaseURL, "The PlanetScale base API URL.")
	return cmd
}

func deleteAccessToken() error {
	tokenPath, err := config.AccessTokenPath()
	if err != nil {
		return err
	}

	err = os.Remove(tokenPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "error removing access token file")
		}
	}

	configFile, err := config.DefaultConfigPath()
	if err != nil {
		return err
	}

	err = os.Remove(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "error removing default config file")
		}
	}

	return nil
}

func waitForEnter(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	return scanner.Err()
}
