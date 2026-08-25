package token

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "show a service token in the organization",
		Args:  cmdutil.RequiredArgs("id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			id := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching service token %s from org %s",
				printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization)))
			defer end()

			token, err := client.ServiceTokens.Get(ctx, &planetscale.GetServiceTokenRequest{
				Organization: ch.Config.Organization,
				ID:           id,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("service token %s does not exist in organization %s",
						printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toServiceTokenDetails(token))
		},
	}

	return cmd
}
