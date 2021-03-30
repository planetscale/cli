package deployrequest

import (
	"errors"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// CloseCmd is the command for closing deploy requests.
func CloseCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close <database> <number>",
		Short: "Close deploy requests",
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
