package deployrequest

import (
	"errors"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// DeployCmd is the command for deploying deploy requests.
func DeployCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy <database> <number>",
		Short: "Deploy a specific deploy request by its number",
		Args:  cmdutil.RequiredArgs("database", "number"),
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
