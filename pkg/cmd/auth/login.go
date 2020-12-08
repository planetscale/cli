package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/planetscale/cli/cmdutil"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

const (
	// TODO(iheanyi): Make this nicer and more cleanly asbtractable, should
	// probably be wrapped in a client and also have the OAuth ClientID be
	// overrideable as a config setting.
	deviceCodeURL = "https://planetscale.us.auth0.com/oauth/device/code"
	oauthTokenURL = "https://planetscale.us.auth0.com/oauth/token"
	oauthClientID = "ZK3V2a5UERfOlWxi5xRXrZZFmvhnf1vg"
)

// DeviceCodeResponse encapsulates the response for obtaining a device code.
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationCompleteURI string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	PollingInterval         int    `json:"interval"`
}

// LoginCmd is the command for logging into a PlanetScale account.
func LoginCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "login",
		Args:    cobra.ExactArgs(0),
		Short:   "Authenticate with PlanetScale",
		Long:    "TODO",
		Example: "TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			fmt.Println("Authenticating")
			payload := strings.NewReader("client_id=ZK3V2a5UERfOlWxi5xRXrZZFmvhnf1vg&scope=profile,email,read:databases,write:databases&audience=https://bb-test-api.planetscale.com")

			req, err := http.NewRequest("POST", deviceCodeURL, payload)
			if err != nil {
				return err
			}

			req = req.WithContext(ctx)
			req.Header.Add("content-type", "application/x-www-form-urlencoded")

			// TODO(iheanyi): Use a better HTTP client than the default one.
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}

			defer res.Body.Close()
			deviceCodeRes := &DeviceCodeResponse{}
			err = json.NewDecoder(res.Body).Decode(deviceCodeRes)
			if err != nil {
				return errors.Wrap(err, "error decoding device code response")
			}

			fmt.Println("Press Enter to authenticate via your browser...")
			_ = waitForEnter(cmd.InOrStdin())
			openCmd := cmdutil.OpenBrowser(runtime.GOOS, deviceCodeRes.VerificationCompleteURI)
			err = openCmd.Run()
			if err != nil {
				return errors.Wrap(err, "error opening browser")
			}

			fmt.Printf("Confirmation Code: %s\n", deviceCodeRes.UserCode)

			// TODO(iheanyi): Revisit why the OAuth login doesn't work as expected.
			accessToken, err := fetchAccessToken(ctx, deviceCodeRes.DeviceCode, deviceCodeRes.PollingInterval, deviceCodeRes.ExpiresIn)
			if err != nil {
				return err
			}

			err = writeAccessToken(ctx, accessToken)
			if err != nil {
				return errors.Wrap(err, "error logging in")
			}
			fmt.Println("Authentication complete.")
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

func fetchAccessToken(ctx context.Context, deviceCode string, pollingInterval int, expiresIn int) (string, error) {
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	checkInterval := time.Duration(pollingInterval) * time.Second
	var accessToken string
	var err error

	for {
		time.Sleep(checkInterval)
		accessToken, err = requestToken(ctx, deviceCode)
		if accessToken == "" && err == nil {
			if time.Now().After(expiresAt) {
				err = errors.New("authentication timed out")
			} else {
				continue
			}
		}

		break
	}

	return accessToken, err
}

// OAuthTokenResponse contains the information returned after fetching an access
// token for a device.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func requestToken(ctx context.Context, deviceCode string) (string, error) {
	payload := strings.NewReader(fmt.Sprintf("grant_type=urn%%3Aietf%%3Aparams%%3Aoauth%%3Agrant-type%%3Adevice_code&device_code=%s&client_id=%s", deviceCode, oauthClientID))
	req, err := http.NewRequest("POST", oauthTokenURL, payload)
	if err != nil {
		return "", errors.Wrap(err, "error creating request")
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	req = req.WithContext(ctx)

	// TODO(iheanyi): Use a better HTTP client than the default one.
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "error performing http request")
	}

	defer res.Body.Close()

	tokenRes := &OAuthTokenResponse{}

	err = json.NewDecoder(res.Body).Decode(tokenRes)
	if err != nil {
		return "", errors.Wrap(err, "error decoding token response")
	}

	return tokenRes.AccessToken, nil
}

func waitForEnter(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	return scanner.Err()
}
