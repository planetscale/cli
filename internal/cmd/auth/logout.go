package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/planetscale/cli/internal/auth"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/spf13/cobra"
)

func LogoutCmd(ch *cmdutil.Helper) *cobra.Command {
	var clientID string
	var clientSecret string
	var apiURL string

	cmd := &cobra.Command{
		Use:   "logout",
		Args:  cobra.NoArgs,
		Short: "Log out of the PlanetScale API",
		RunE: func(cmd *cobra.Command, args []string) error {
			// An unreadable token can't be revoked, but deleting it still logs out.
			readErr := ch.Config.LoadStoredAccessToken()
			if readErr == nil && ch.Config.AccessToken == "" {
				ch.Printer.Println("Already logged out. Exiting...")
				return nil
			}

			if printer.IsTTY {
				ch.Printer.Println("Press Enter to log out of the PlanetScale API.")
				_ = waitForEnter(cmd.InOrStdin())
			}

			clientID, clientSecret = resolveOAuthClient(clientID, clientSecret)
			authenticator, err := auth.New(cleanhttp.DefaultClient(), clientID, clientSecret, auth.SetBaseURL(apiURL))
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			end := ch.Printer.PrintProgress("Logging out...")
			defer end()
			if readErr == nil {
				err = authenticator.RevokeToken(ctx, ch.Config.AccessToken)
				if err != nil {
					return err
				}
			}
			err = config.DeleteAccessToken()
			if err != nil {
				return errors.Join(readErr, err)
			}
			end()
			if readErr != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: deleted the stored access token without revoking it: %v\n", readErr)
			}
			ch.Printer.Println("Successfully logged out.")

			return nil
		},
	}

	addOAuthClientFlags(cmd, &clientID, &clientSecret)
	cmd.Flags().StringVar(&apiURL, "api-url", auth.DefaultBaseURL, "The PlanetScale base API URL.")
	return cmd
}

func waitForEnter(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	return scanner.Err()
}
