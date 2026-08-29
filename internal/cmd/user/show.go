package user

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the currently authenticated user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress("Fetching current user")
			defer end()

			user, err := client.Users.GetCurrentUser(ctx)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()

			return ch.Printer.PrintResource(toUser(user))
		},
	}

	return cmd
}
