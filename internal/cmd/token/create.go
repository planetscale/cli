package token

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create a service token for the organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := ch.Client()
			if err != nil {
				return err
			}

			req := &planetscale.CreateServiceTokenRequest{
				Organization: ch.Config.Organization,
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating service token in org %s", printer.BoldBlue(ch.Config.Organization)))
			defer end()

			token, err := client.ServiceTokens.Create(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("organization %s does not exist", printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			return ch.Printer.PrintResource(toServiceToken(token))
		},
	}

	return cmd
}
