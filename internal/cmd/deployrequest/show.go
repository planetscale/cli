package deployrequest

import (
	"errors"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// ShowCmd is the command to show a deploy request.
func ShowCmd(cfg *config.Config) *cobra.Command {
	var flags struct {
		web bool
	}

	cmd := &cobra.Command{
		Use:   "show <database> <number>",
		Short: "Show a specific deploy request",
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
