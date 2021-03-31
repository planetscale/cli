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
)

func LogoutCmd(cfg *config.Config) *cobra.Command {
	var clientID string
	var clientSecret string
	var apiURL string

	cmd := &cobra.Command{
		Use:   "logout",
		Args:  cobra.NoArgs,
		Short: "Log out of the PlanetScale API",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.AccessToken == "" {
				fmt.Println("Already logged out. Exiting...")
				return nil
			}

			if cmdutil.IsTTY {
				fmt.Println("Press Enter to log out of the PlanetScale API.")
				_ = waitForEnter(cmd.InOrStdin())
			}

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
	tokenPath, err := config.AccessTokenPath()
	if err != nil {
		return err
	}

	err = os.Remove(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("yooo tokenpath")
		}
		fmt.Printf("err = %+v\n", err)
		return errors.Wrap(err, "error removing access token file")
	}

	configFile, err := config.DefaultConfigPath()
	if err != nil {
		return err
	}

	err = os.Remove(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("yooo configFile")
		}
		return errors.Wrap(err, "error removing default config file")
	}

	return nil
}
