package auth

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/planetscale/cli/internal/auth"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/fatih/color"
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
		Short: "Authenticate with the PlanetScale API",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !printer.IsTTY {
				return errors.New("the 'login' command requires an interactive shell")
			}

			authenticator, err := auth.New(cleanhttp.DefaultClient(), clientID, clientSecret, auth.SetBaseURL(authURL))
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			deviceVerification, err := authenticator.VerifyDevice(ctx)
			if err != nil {
				return err
			}

			openCmd := cmdutil.OpenBrowser(runtime.GOOS, deviceVerification.VerificationCompleteURL)
			err = openCmd.Run()
			if err != nil {
				ch.Printer.Printf("Failed to open a browser: %s\n", printer.BoldRed(err.Error()))
			}

			bold := color.New(color.Bold)
			bold.Printf("\nConfirmation Code: ")
			boldGreen := bold.Add(color.FgGreen)
			boldGreen.Fprintln(color.Output, deviceVerification.UserCode)

			ch.Printer.Printf("\nIf something goes wrong, copy and paste this URL into your browser: %s\n\n", printer.Bold(deviceVerification.VerificationCompleteURL))

			end := ch.Printer.PrintProgress("Waiting for confirmation...")
			defer end()
			accessToken, err := authenticator.GetAccessTokenForDevice(ctx, *deviceVerification)
			if err != nil {
				return err
			}

			err = config.WriteAccessToken(accessToken)
			if err != nil {
				return fmt.Errorf("error logging in: %w", err)
			}

			// We explicitly stop here so we can replace the spinner with our success
			// message.
			end()
			ch.Printer.Println("Successfully logged in.")

			writeConfig := false
			cfg, err := ch.ConfigFS.DefaultConfig()
			if err != nil {
				writeConfig = true
			}

			if !writeConfig && cfg.Organization != "" {
				hasOrg, _ := hasOrg(ctx, cfg.Organization, accessToken, authURL)
				writeConfig = !hasOrg
			}

			if writeConfig || cfg.Organization == "" {
				err = writeDefaultOrganization(ctx, accessToken, authURL)
				if err != nil {
					return err
				}
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
	orgs, err := listCurrentOrgs(ctx, accessToken, authURL)
	if err != nil {
		return err
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

func hasOrg(ctx context.Context, org, accessToken, authURL string) (bool, error) {
	currentOrgs, err := listCurrentOrgs(ctx, accessToken, authURL)
	if err != nil {
		return false, err
	}

	for _, o := range currentOrgs {
		if o.Name == org {
			return true, nil
		}
	}

	return false, nil
}

func listCurrentOrgs(ctx context.Context, accessToken, authURL string) ([]*planetscale.Organization, error) {
	client, err := planetscale.NewClient(
		planetscale.WithAccessToken(accessToken),
		planetscale.WithBaseURL(authURL),
	)
	if err != nil {
		return nil, cmdutil.HandleError(err)
	}

	orgs, err := client.Organizations.List(ctx)
	if err != nil {
		return nil, cmdutil.HandleError(err)
	}

	return orgs, nil

}
