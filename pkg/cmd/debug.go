package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

const ()

func DebugCmd(cfg *config.Config) *cobra.Command {
	var accessToken string
	var apiURL string
	cmd := &cobra.Command{
		Use:   "debug",
		Args:  cobra.ExactArgs(0),
		Short: "Check if authorized API calls work",
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := http.NewRequest("GET", apiURL, nil)
			if err != nil {
				return err
			}

			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cfg.AccessToken))
			req.Header.Add("content-type", "application/json")

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}
			fmt.Println(string(body))
			fmt.Println(res.StatusCode)
			return nil
		},
	}

	cmd.Flags().StringVar(&accessToken, "access-token", cfg.AccessToken, "The access token for communicating with the API")
	cmd.Flags().StringVar(&apiURL, "api-url", "", "The PlanetScale base API URL.")
	return cmd
}
