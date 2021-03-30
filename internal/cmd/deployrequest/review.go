package deployrequest

import (
	"errors"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

// ReviewCmd is the command for reviewing (approve, comment, etc.) a deploy
// request.
func ReviewCmd(cfg *config.Config) *cobra.Command {
	var flags struct {
		approve bool
		comment string
	}

	cmd := &cobra.Command{
		Use:   "review <database> <number>",
		Short: "Review a deploy request (approve, comment, etc...)",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !flags.approve && flags.comment == "" {
				return errors.New("neither --approve nor --comment is set")
			}

			_, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			return errors.New("not implemented yet")
		},
	}

	cmd.PersistentFlags().BoolVar(&flags.approve, "approve", false, "Approve a deploy request")
	cmd.PersistentFlags().StringVar(&flags.comment, "comment", "", "Comment on a deploy request")

	return cmd
}
