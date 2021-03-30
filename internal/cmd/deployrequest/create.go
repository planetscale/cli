package deployrequest

import (
	"errors"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// CreateCmd is the command for creating deploy requests.
func CreateCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a deploy request from a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			return errors.New("not implemented yet")
		},
	}

	return cmd
}
