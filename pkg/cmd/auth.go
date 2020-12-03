package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/cli/safeexec"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	deviceCodeURL     = "https://planetscale.us.auth0.com/oauth/device/code"
	oauthTokenURL     = "https://planetscale.us.auth0.com/oauth/token"
	oauthClientID     = "ZK3V2a5UERfOlWxi5xRXrZZFmvhnf1vg"
	defaultConfigPath = "~/.config/psctl"
)

// DeviceCodeResponse encapsulates the response for obtaining a device code.
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationCompleteURI string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	PollingInterval         int    `json:"interval"`
}

// ErrorResponse wraps the error from the authorization API.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// AuthCmd returns a command for authentication
func AuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth <command>",
		Short: "Login, logout, and refresh your authentication",
		Long:  "Manage psctl's Cauthentication state.",
	}

	cmd.AddCommand(LoginCmd())
	return cmd
}

// LoginCmd is the command for logging into a PlanetScale account.
func LoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "login",
		Args:    cobra.ExactArgs(0),
		Short:   "Authenticate with PlanetScale",
		Long:    "TODO",
		Example: "TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			fmt.Println("Authenticating")
			payload := strings.NewReader("client_id=ZK3V2a5UERfOlWxi5xRXrZZFmvhnf1vg&scope=profile&audience=https://bb-test-api.planetscale.com")

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
			json.NewDecoder(res.Body).Decode(deviceCodeRes)

			fmt.Println("Press Enter to authenticate via your browser...")
			_ = waitForEnter(cmd.InOrStdin())
			openCmd := OpenBrowserCmd(runtime.GOOS, deviceCodeRes.VerificationCompleteURI)
			err = openCmd.Run()
			if err != nil {
				return errors.Wrap(err, "error opening browser")
			}

			// TODO(iheanyi): Revisit why the OAuth login doesn't work as expected.
			accessToken, err := fetchAccessToken(ctx, deviceCodeRes.DeviceCode, deviceCodeRes.PollingInterval, deviceCodeRes.ExpiresIn)
			if err != nil {
				return err
			}

			// TODO(iheanyi): Write file here
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

func configDir() string {
	dir, _ := homedir.Expand(defaultConfigPath)
	return dir
}

func accessTokenPath() string {
	return path.Join(configDir(), "access-token")
}

func writeAccessToken(ctx context.Context, accessToken string) error {
	_, err := os.Stat(configDir())
	if os.IsNotExist(err) {
		err := os.MkdirAll(configDir(), 0600)
		if err != nil {
			return errors.Wrap(err, "error creating config directory")
		}
	} else {
		return err
	}

	tokenBytes := []byte(accessToken)
	err = ioutil.WriteFile(accessTokenPath(), tokenBytes, 0666)
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

	req.WithContext(ctx)

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

func linuxExe() string {
	exe := "xdg-open"

	_, err := lookPath(exe)
	if err != nil {
		_, err := lookPath("wslview")
		if err == nil {
			exe = "wslview"
		}
	}

	return exe
}

// OpenBrowserCmd opens a browser at the inputted URL.
func OpenBrowserCmd(goos, url string) *exec.Cmd {
	exe := "open"
	var args []string
	switch goos {
	case "darwin":
		args = append(args, url)
	case "windows":
		exe, _ = lookPath("cmd")
		r := strings.NewReplacer("&", "^&")
		args = append(args, "/c", "start", r.Replace(url))
	default:
		exe = linuxExe()
		args = append(args, url)
	}
	cmd := exec.Command(exe, args...)
	cmd.Stderr = os.Stderr
	return cmd
}

func waitForEnter(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	return scanner.Err()
}

var lookPath = safeexec.LookPath
