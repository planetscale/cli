package auth

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/pkg/errors"
	"github.com/planetscale/cli/cmdutil"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

const (
	logoutURL = "https://planetscale.us.auth0.com/v2/logout"
)

func LogoutCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logout",
		Args:    cobra.ExactArgs(0),
		Short:   "Log the user out",
		Long:    "TODO",
		Example: "TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Press Enter to logout via browser...")
			_ = waitForEnter(cmd.InOrStdin())
			openCmd := cmdutil.OpenBrowser(runtime.GOOS, logoutURL)
			err := openCmd.Run()
			if err != nil {
				return errors.Wrap(err, "error opening browser")
			}

			fmt.Println("Successfully logged out.")

			return nil
		},
	}

	return cmd
}

func deleteAccessToken(ctx context.Context, accessToken string) error {
	_, err := os.Stat(config.AccessTokenPath())
	if os.IsNotExist(err) {
		return nil
	}

	err = os.Remove(config.AccessTokenPath())
	if err != nil {
		return errors.Wrap(err, "error removing file")
	}

	return nil
}
