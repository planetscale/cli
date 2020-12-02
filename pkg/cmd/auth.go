package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/cli/safeexec"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DeviceCodeResponse encapsulates the response for obtaining a device code.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	PollingInterval int    `json:"interval"`
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

func LoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "login",
		Args:    cobra.ExactArgs(0),
		Short:   "Authenticate with PlanetScale",
		Long:    "TODO",
		Example: "TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Authenticating")
			url := "https://planetscale.us.auth0.com/oauth/device/code"
			payload := strings.NewReader("client_id=ZK3V2a5UERfOlWxi5xRXrZZFmvhnf1vg&scope=profile,email&audience=https://bb-test-api.planetscale.com")

			req, err := http.NewRequest("POST", url, payload)
			if err != nil {
				return err
			}

			req.Header.Add("content-type", "application/x-www-form-urlencoded")

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}

			defer res.Body.Close()
			deviceCodeRes := &DeviceCodeResponse{}
			json.NewDecoder(res.Body).Decode(deviceCodeRes)

			fmt.Println("Please enter the following code in the web browser.")
			openCmd := OpenBrowserCmd(runtime.GOOS, deviceCodeRes.VerificationURI)
			err = openCmd.Run()
			if err != nil {
				errors.Wrap(err, "error opening browser")
			}
			fmt.Printf("%v\n", deviceCodeRes)
			return nil
		},
	}

	return cmd
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
	fmt.Println(exe)
	cmd := exec.Command(exe, args...)
	cmd.Stderr = os.Stderr
	return cmd
}

var lookPath = safeexec.LookPath
