package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/planetscale/cli/internal/auth"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/fatih/color"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// LoginCmd is the command for logging into a PlanetScale account.
func LoginCmd(ch *cmdutil.Helper) *cobra.Command {
	var clientID string
	var clientSecret string
	var authURL string

	cmd := &cobra.Command{
		Use:   "login",
		Args:  cobra.ExactArgs(0),
		Short: "Authenticate with PlanetScale",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !printer.IsTTY {
				return errors.New("The 'login' command requires an interactive shell")
			}

			authenticator, err := auth.New(cleanhttp.DefaultClient(), clientID, clientSecret, auth.SetBaseURL(authURL))
			if err != nil {
				return err
			}
			ctx := context.Background()

			deviceVerification, err := authenticator.VerifyDevice(ctx)
			if err != nil {
				return err
			}

			openCmd := cmdutil.OpenBrowser(runtime.GOOS, deviceVerification.VerificationCompleteURL)
			err = openCmd.Run()
			if err != nil {
				return errors.Wrap(err, "error opening browser")
			}

			bold := color.New(color.Bold)
			bold.Printf("Confirmation Code: ")
			boldGreen := bold.Add(color.FgGreen)
			boldGreen.Println(deviceVerification.UserCode)

			fmt.Printf("\nIf something goes wrong, copy and paste this URL into your browser: %s\n\n", printer.Bold(deviceVerification.VerificationCompleteURL))

			end := ch.Printer.PrintProgress("Waiting for confirmation...")
			defer end()
			accessToken, err := authenticator.GetAccessTokenForDevice(ctx, deviceVerification)
			if err != nil {
				return err
			}

			err = writeAccessToken(ctx, accessToken)
			if err != nil {
				return errors.Wrap(err, "error logging in")
			}

			// We explicitly stop here so we can replace the spinner with our success
			// message.
			end()
			ch.Printer.Println("Successfully logged in.")

			err = writeDefaultOrganization(ctx, accessToken, authURL)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", auth.OAuthClientID, "The client ID for the PlanetScale CLI application.")
	cmd.Flags().StringVar(&clientSecret, "client-secret", auth.OAuthClientSecret, "The client ID for the PlanetScale CLI application")
	cmd.Flags().StringVar(&authURL, "api-url", auth.DefaultBaseURL, "The PlanetScale Auth API base URL.")

	return cmd
}

func writeDefaultOrganization(ctx context.Context, accessToken, authURL string) error {
	// After successfully logging in, attempt to set the org by default.
	client, err := planetscale.NewClient(
		planetscale.WithAccessToken(accessToken),
		planetscale.WithBaseURL(authURL),
	)
	if err != nil {
		return err
	}

	orgs, err := client.Organizations.List(ctx)
	if err != nil {
		return cmdutil.HandleError(err)
	}

	if len(orgs) > 0 {
		defaultOrg := orgs[0].Name
		writableConfig := &config.FileConfig{
			Organization: defaultOrg,
		}

		err := writableConfig.WriteDefault()
		if err != nil {
			return err
		}
	}

	return nil
}

func writeAccessToken(ctx context.Context, accessToken string) error {
	configDir, err := config.ConfigDir()
	if err != nil {
		return err
	}

	_, err = os.Stat(configDir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(configDir, 0771)
		if err != nil {
			return errors.Wrap(err, "error creating config directory")
		}
	} else if err != nil {
		return err
	}

	tokenPath, err := config.AccessTokenPath()
	if err != nil {
		return err
	}

	tokenBytes := []byte(accessToken)
	err = ioutil.WriteFile(tokenPath, tokenBytes, config.TokenFileMode)
	if err != nil {
		return errors.Wrap(err, "error writing token")
	}

	return nil
}
