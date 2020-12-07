package auth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"

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
	cmd := &cobra.Command{
		Use:     "login",
		Args:    cobra.ExactArgs(0),
		Short:   "Authenticate with PlanetScale",
		Long:    "TODO",
		Example: "TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			authenticator, err := auth.New(cleanhttp.DefaultClient())
			if err != nil {
				return err
			}
			ctx := context.Background()

			fmt.Println("Authenticating...")
			deviceVerification, err := authenticator.VerifyDevice(ctx, auth.DefaultOAuthClientID, auth.DefaultAudienceURL)
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

			accessToken, err := authenticator.GetAccessTokenForDevice(ctx, deviceVerification, auth.DefaultOAuthClientID)
			if err != nil {
				return err
			}

			err = writeAccessToken(ctx, accessToken)
			if err != nil {
				return errors.Wrap(err, "error logging in")
			}
			fmt.Println("Successfully logged in!")
			return nil
		},
	}

	return cmd
}

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
