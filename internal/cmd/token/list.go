package token

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list service tokens for the organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := ch.Client()
			if err != nil {
				return err
			}

			req := &planetscale.ListServiceTokensRequest{
				Organization: ch.Config.Organization,
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating service token in org %s", printer.BoldBlue(ch.Config.Organization)))
			defer end()

			tokens, err := client.ServiceTokens.List(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("organization %s does not exist\n", printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return errors.Wrap(err, "error listing service tokens")
				}
			}

			end()

			return ch.Printer.PrintResource(toServiceTokens(tokens))
		},
	}

	return cmd
}
