package token

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-access <name>",
		Short: "fetch a service token and it's accesses",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) != 1 {
				return cmd.Usage()
			}

			name := args[0]

			req := &planetscale.GetServiceTokenAccessRequest{
				ID:           name,
				Organization: ch.Config.Organization,
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating service token in org %s", printer.BoldBlue(ch.Config.Organization)))
			defer end()

			accesses, err := client.ServiceTokens.GetAccess(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("access %s does not exist in organization %s\n",
						printer.BoldBlue(name), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()

			return ch.Printer.PrintResource(toServiceTokenAccesses(accesses))
		},
	}

	return cmd
}
