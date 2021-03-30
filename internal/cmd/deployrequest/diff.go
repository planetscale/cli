package deployrequest

import (
	"errors"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// DiffCmd is the command for showing the diff of a deploy request.
func DiffCmd(cfg *config.Config) *cobra.Command {
	var flags struct {
		web bool
	}

	cmd := &cobra.Command{
		Use:   "diff <database> <number>",
		Short: "Show the diff of a deploy request",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.web {
				fmt.Println("🌐  Redirecting you to your deploy request in your web browser.")
				// TODO(fatih): immplement
				return nil
			}

			_, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			return errors.New("not implemented yet")
		},
	}

	cmd.PersistentFlags().BoolVar(&flags.web, "web", false, "Open in your web browser")

	return cmd
}
