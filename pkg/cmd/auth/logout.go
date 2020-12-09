package auth

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

func LogoutCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logout",
		Args:    cobra.ExactArgs(0),
		Short:   "Log out of the PlanetScale API",
		Long:    "TODO",
		Example: "TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Press Enter to log out of the PlanetScale API.")
			_ = waitForEnter(cmd.InOrStdin())

			err := deleteAccessToken()
			if err != nil {
				return err
			}
			fmt.Println("Successfully logged out.")

			return nil
		},
	}

	return cmd
}

func deleteAccessToken() error {
	_, err := os.Stat(config.AccessTokenPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	err = os.Remove(config.AccessTokenPath())
	if err != nil {
		return errors.Wrap(err, "error removing file")
	}

	return nil
}
