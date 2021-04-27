package signup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

type registerRequest struct {
	User registerUser `json:"user"`
}

type registerUser struct {
	Email                string `json:"email" survey:"email"`
	Password             string `json:"password" survey:"password"`
	PasswordConfirmation string `json:"password_confirmation" survey:"password_confirmation"`
}

type registerResponse struct {
	ID        int
	PublidID  string
	Email     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

// SignupCmd encapsulates the command for signing up to a new PlanetScale account.
func SignupCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "signup <action>",
		Short:   "Signup for a new PlanetScale account",
		Aliases: []string{"register", "sign-up"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !printer.IsTTY {
				return errors.New("pscale signup only works in interactive mode")
			}

			ch.Printer.Println("You are registering a new PlanetScale account.")

			var qs = []*survey.Question{
				{
					Name:     "email",
					Prompt:   &survey.Input{Message: "What is your e-mail?"},
					Validate: survey.ComposeValidators(survey.Required),
				},
				{
					Name: "password",
					Prompt: &survey.Password{
						Message: "Please type your password",
					},
					Validate: survey.Required,
				},
				{
					Name: "password_confirmation",
					Prompt: &survey.Password{
						Message: "Please confirm your password",
					},
					Validate: survey.Required,
				},
			}

			user := registerUser{}
			err := survey.Ask(qs, &user)
			if err != nil {
				return err
			}

			params := &registerRequest{User: user}
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(params); err != nil {
				return err
			}

			path := fmt.Sprintf("%sinternal/register", planetscale.DefaultBaseURL)
			req, err := http.NewRequest("POST", path, &buf)
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")

			client := &http.Client{Timeout: time.Second * 15}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			out, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			success := resp.StatusCode >= 200 && resp.StatusCode < 300
			if !success {
				type errorResponse struct {
					Errors struct {
						Email                []string `json:"email"`
						Password             []string `json:"password"`
						PasswordConfirmation []string `json:"password_confirmation"`
					} `json:"errors"`
				}

				res := &errorResponse{}
				if err := json.Unmarshal(out, res); err != nil {
					return err
				}

				errMsg := ""
				if len(res.Errors.Email) != 0 {
					errMsg += fmt.Sprintf("%s: %s\n", printer.BoldRed("email error"), strings.Join(res.Errors.Email, ","))
				}
				if len(res.Errors.Password) != 0 {
					errMsg += fmt.Sprintf("%s: %s\n", printer.BoldRed("password error"), strings.Join(res.Errors.Password, ","))
				}
				if len(res.Errors.PasswordConfirmation) != 0 {
					errMsg += fmt.Sprintf("%s: %s", printer.BoldRed("password error"), strings.Join(res.Errors.PasswordConfirmation, ","))
				}

				return fmt.Errorf("error registering user\n\n%v", errMsg)
			}

			// TODO(fatih): should we do anythying with the response?
			var r *registerResponse
			err = json.Unmarshal(out, &r)
			if err != nil {
				return err
			}

			ch.Printer.Println("\nYou've successfully signed up for PlanetScale!\nPlease check your email for a confirmation link and then get started with `pscale auth login`. ")

			return nil

		},
	}

	return cmd
}
