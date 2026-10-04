package auth

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/hashicorp/go-cleanhttp"
	psauth "github.com/planetscale/cli/internal/auth"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const savingCredentialsMessage = "Access approved. Saving credentials to your system keyring; if it asks to be unlocked, enter your keyring password..."

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
			jsonMode := ch.Printer.Format() == printer.JSON
			if !jsonMode && !printer.IsTTY {
				return errors.New("the 'login' command requires an interactive shell (use --format json; browser opens when possible, then polls until approved)")
			}

			clientID, clientSecret = resolveOAuthClient(clientID, clientSecret)
			authenticator, err := psauth.New(cleanhttp.DefaultClient(), clientID, clientSecret, psauth.SetBaseURL(authURL))
			if err != nil {
				if jsonMode {
					return finishLoginErrorJSON(ch, "AUTH_INIT_FAILED", "Failed to initialize login", err)
				}
				return err
			}

			ctx := cmd.Context()

			deviceVerification, err := authenticator.VerifyDevice(ctx)
			if err != nil {
				if jsonMode {
					return finishLoginErrorJSON(ch, "DEVICE_VERIFY_FAILED", "Failed to start device authorization", err)
				}
				return err
			}

			browserOpened := cmdutil.TryOpenBrowser(runtime.GOOS, deviceVerification.VerificationCompleteURL) == nil

			if jsonMode {
				pending := LoginPendingResponse{
					Status:          "pending",
					VerificationURL: deviceVerification.VerificationCompleteURL,
					UserCode:        deviceVerification.UserCode,
					BrowserOpened:   browserOpened,
					Message:         loginPendingMessage(browserOpened),
					NextSteps: []string{
						"Approve access in the browser",
						cmdutil.AgentAuthCheckCmd(),
					},
				}
				if err := printJSONEnvelope(cmd.ErrOrStderr(), pending); err != nil {
					return err
				}
			} else {
				if !browserOpened {
					ch.Printer.Printf("Failed to open a browser; open this URL manually: %s\n", printer.Bold(deviceVerification.VerificationCompleteURL))
				}

				bold := color.New(color.Bold)
				bold.Printf("\nConfirmation Code: ")
				boldGreen := bold.Add(color.FgGreen)
				boldGreen.Fprintln(color.Output, deviceVerification.UserCode)

				ch.Printer.Printf("\nIf something goes wrong, copy and paste this URL into your browser: %s\n\n", printer.Bold(deviceVerification.VerificationCompleteURL))
			}

			var progress *printer.ProgressHandle
			if !jsonMode {
				progress = ch.Printer.StartProgress("Waiting for confirmation...")
				defer progress.Stop()
			}

			accessToken, err := authenticator.GetAccessTokenForDevice(ctx, *deviceVerification)
			if err != nil {
				if jsonMode {
					return finishLoginErrorJSON(ch, "DEVICE_AUTH_FAILED", "Device authorization failed or timed out", err)
				}
				return err
			}

			// Saving to the system keyring can block on an unlock prompt,
			// such as GNOME Keyring's password dialog, which may open on
			// another screen. Without this, login looks stuck waiting for the
			// browser even though access was already approved.
			progress.Update(savingCredentialsMessage)

			err = config.WriteAccessToken(accessToken)
			if err != nil {
				if jsonMode {
					return finishLoginErrorJSON(ch, "TOKEN_SAVE_FAILED", "Failed to save access token", err)
				}
				configDir, configErr := config.ConfigDir()
				if configErr != nil {
					ch.Printer.Printf("Error looking up configuration directory: %s\n", printer.BoldRed(configErr.Error()))
				}
				return fmt.Errorf("error logging in: %w\n\nPlease ensure you have write permissions to the configuration directory: %s", err, configDir)
			}

			progress.Stop()

			orgSetupErr := writeDefaultOrganizationIfNeeded(ctx, ch, accessToken, authURL)

			if jsonMode {
				return finishLoginJSON(ch, orgSetupErr)
			}

			if orgSetupErr != nil {
				return orgSetupErr
			}

			ch.Printer.Println("Successfully logged in.")
			return nil
		},
	}

	addOAuthClientFlags(cmd, &clientID, &clientSecret)
	cmd.Flags().StringVar(&authURL, "api-url", psauth.DefaultBaseURL, "The PlanetScale Auth API base URL.")

	return cmd
}

func finishLoginJSON(ch *cmdutil.Helper, orgSetupErr error) error {
	resp := LoginOKResponse{
		NextSteps: []string{
			cmdutil.AgentAuthCheckCmd(),
			cmdutil.AgentOrgListCmd(),
		},
	}
	if orgSetupErr != nil {
		resp.Status = "action_required"
		resp.Message = "Credentials saved, but organization setup failed."
		resp.Issues = []AuthIssue{{
			Code:        "ORG_SETUP_FAILED",
			Message:     orgSetupErr.Error(),
			Remediation: "Credentials were saved; run `pscale org list --format json` and `pscale org switch <org>`",
		}}
		resp.NextSteps = []string{
			cmdutil.AgentOrgListCmd(),
			cmdutil.AgentAuthCheckCmd(),
		}
	} else {
		resp.Status = "ok"
		resp.Message = "Successfully logged in."
	}
	if err := ch.Printer.PrintJSON(resp); err != nil {
		return err
	}
	if resp.Status == "action_required" {
		return cmdutil.JSONReportedError(cmdutil.ActionRequestedExitCode)
	}
	return nil
}

func writeDefaultOrganizationIfNeeded(ctx context.Context, ch *cmdutil.Helper, accessToken, authURL string) error {
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
		return writeDefaultOrganization(ctx, accessToken, authURL)
	}
	return nil
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
