package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

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
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}

			fmt.Println(res)
			fmt.Println(string(body))
			return nil
		},
	}

	return cmd
}
