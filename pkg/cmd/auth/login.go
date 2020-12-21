package auth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	"github.com/planetscale/cli/auth"
	"github.com/planetscale/cli/cmdutil"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// LoginCmd is the command for logging into a PlanetScale account.
func LoginCmd(cfg *config.Config) *cobra.Command {
	var clientID string
	var clientSecret string
	var apiURL string

	cmd := &cobra.Command{
		Use:     "login",
		Args:    cobra.ExactArgs(0),
		Short:   "Authenticate with PlanetScale",
		Long:    "TODO",
		Example: "TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			authenticator, err := auth.New(cleanhttp.DefaultClient(), auth.OAuthClientID, auth.OAuthClientSecret, auth.SetBaseURL(apiURL))
			if err != nil {
				return err
			}
			ctx := context.Background()

			deviceVerification, err := authenticator.VerifyDevice(ctx)
			if err != nil {
				return err
			}

			fmt.Println("Press Enter to authenticate via your browser...")

			_ = waitForEnter(cmd.InOrStdin())
			openCmd := cmdutil.OpenBrowser(runtime.GOOS, deviceVerification.VerificationCompleteURL)
			err = openCmd.Run()
			if err != nil {
				return errors.Wrap(err, "error opening browser")
			}

			bold := color.New(color.Bold)
			bold.Printf("Confirmation Code: ")
			boldGreen := bold.Add(color.FgGreen)
			boldGreen.Println(deviceVerification.UserCode)

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			s.Suffix = " Waiting for confirmation..."
			err = s.Color("bold", "green")
			if err != nil {
				return errors.Wrap(err, "error setting color")
			}

			s.Start()
			defer s.Stop()
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
			s.Stop()
			fmt.Println("Successfully logged in!")
			return nil
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", auth.OAuthClientID, "The client ID for the PlanetScale CLI application.")
	cmd.Flags().StringVar(&clientSecret, "client-secret", auth.OAuthClientSecret, "The client ID for the PlanetScale CLI application")
	cmd.Flags().StringVar(&apiURL, "api-url", auth.DefaultBaseURL, "The PlanetScale base API URL.")

	return cmd
}

// TODO(iheanyi): Double-check the file permissions in this function.
func writeAccessToken(ctx context.Context, accessToken string) error {
	_, err := os.Stat(config.ConfigDir())
	if os.IsNotExist(err) {
		err := os.MkdirAll(config.ConfigDir(), 0771)
		if err != nil {
			return errors.Wrap(err, "error creating config directory")
		}
	} else if err != nil {
		return err
	}

	tokenBytes := []byte(accessToken)
	err = ioutil.WriteFile(config.AccessTokenPath(), tokenBytes, 0666)
	if err != nil {
		return errors.Wrap(err, "error writing token")
	}

	return nil
}

func waitForEnter(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	return scanner.Err()
}
