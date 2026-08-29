package oauthapplication

import (
	"fmt"
	"io"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func TokenCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		clientID     string
		clientSecret string
		grantType    string
		code         string
		redirectURI  string
		refreshToken string
	}

	cmd := &cobra.Command{
		Use:   "create <application-id>",
		Short: "Create or renew an OAuth application token",
		Args:  cmdutil.RequiredArgs("application-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.grantType != "authorization_code" && flags.grantType != "refresh_token" {
				return fmt.Errorf("--grant-type must be authorization_code or refresh_token")
			}
			if flags.grantType == "authorization_code" {
				if flags.code == "" || flags.redirectURI == "" {
					return fmt.Errorf("--grant-type authorization_code requires --code and --redirect-uri")
				}
				if flags.refreshToken != "" {
					return fmt.Errorf("--grant-type authorization_code cannot be used with --refresh-token")
				}
			} else {
				if flags.refreshToken == "" {
					return fmt.Errorf("--grant-type refresh_token requires --refresh-token")
				}
				if flags.code != "" || flags.redirectURI != "" {
					return fmt.Errorf("--grant-type refresh_token cannot be used with --code or --redirect-uri")
				}
			}

			clientSecret := flags.clientSecret
			if clientSecret == "@-" {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return fmt.Errorf("reading --client-secret from stdin: %w", err)
				}
				clientSecret = strings.TrimSpace(string(data))
			}
			if clientSecret == "" {
				return fmt.Errorf("--client-secret cannot be empty")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}
			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating OAuth token for application %s", printer.BoldBlue(args[0])))
			defer end()
			token, err := client.OAuthApplications.CreateToken(cmd.Context(), &ps.CreateOAuthTokenRequest{
				Organization: ch.Config.Organization,
				ID:           args[0],
				ClientID:     flags.clientID,
				ClientSecret: clientSecret,
				GrantType:    flags.grantType,
				Code:         flags.code,
				RedirectURI:  flags.redirectURI,
				RefreshToken: flags.refreshToken,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("OAuth application %s does not exist in organization %s",
						printer.BoldBlue(args[0]), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()
			if ch.Printer.Format() == printer.Human {
				ch.Printer.Println("The plaintext token and refresh token below are shown only once.")
			}
			return ch.Printer.PrintResource(toOAuthTokenWithSecret(token))
		},
	}

	cmd.Flags().StringVar(&flags.clientID, "client-id", "", "OAuth application client ID")
	cmd.Flags().StringVar(&flags.clientSecret, "client-secret", "", `OAuth application client secret, or "@-" to read it from stdin`)
	cmd.Flags().StringVar(&flags.grantType, "grant-type", "authorization_code", "OAuth grant type: authorization_code or refresh_token")
	cmd.Flags().StringVar(&flags.code, "code", "", "OAuth authorization code")
	cmd.Flags().StringVar(&flags.redirectURI, "redirect-uri", "", "OAuth application redirect URI")
	cmd.Flags().StringVar(&flags.refreshToken, "refresh-token", "", "OAuth refresh token")
	_ = cmd.MarkFlagRequired("client-id")
	_ = cmd.MarkFlagRequired("client-secret")
	return cmd
}
