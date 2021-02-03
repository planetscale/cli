package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/auth"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func LogoutCmd(cfg *config.Config) *cobra.Command {
	var clientID string
	var clientSecret string
	var apiURL string

	cmd := &cobra.Command{
		Use:     "logout",
		Args:    cobra.ExactArgs(0),
		Short:   "Log out of the PlanetScale API",
		Long:    "TODO",
		Example: "TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.AccessToken == "" {
				fmt.Println("Already logged out. Exiting...")
				return nil
			}
			fmt.Println("Press Enter to log out of the PlanetScale API.")
			_ = waitForEnter(cmd.InOrStdin())

			authenticator, err := auth.New(cleanhttp.DefaultClient(), clientID, clientSecret, auth.SetBaseURL(apiURL))
			if err != nil {
				return err
			}
			ctx := context.Background()

			end := cmdutil.PrintProgress("Logging out...")
			defer end()
			err = authenticator.RevokeToken(ctx, cfg.AccessToken)
			if err != nil {
				return err
			}
			err = deleteAccessToken()
			if err != nil {
				return err
			}
			end()
			fmt.Println("Successfully logged out.")

			return nil
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", auth.OAuthClientID, "The client ID for the PlanetScale CLI application.")
	cmd.Flags().StringVar(&clientSecret, "client-secret", auth.OAuthClientSecret, "The client ID for the PlanetScale CLI application")
	cmd.Flags().StringVar(&apiURL, "api-url", auth.DefaultBaseURL, "The PlanetScale base API URL.")
	return cmd
}

func deleteAccessToken() error {
	_, err := os.Stat(config.AccessTokenPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	err = os.Remove(config.AccessTokenPath())
	if err != nil {
		return errors.Wrap(err, "error removing access token file")
	}

	err = os.Remove(viper.ConfigFileUsed())
	if err != nil {
		return errors.Wrap(err, "error removing config file")
	}

	return nil
}
