package token

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ShowAccessCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-access <name>",
		Short: "fetch a service token and it's accesses",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := ch.Client()
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching service token from org %s", printer.BoldBlue(ch.Config.Organization)))
			defer end()

			accesses, err := client.ServiceTokens.GetAccess(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("access %s does not exist in organization %s",
						printer.BoldBlue(name), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			return ch.Printer.PrintResource(toServiceTokenAccesses(accesses))
		},
	}

	return cmd
}
