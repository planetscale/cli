package token

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <token>",
		Short: "delete an entire service token in an organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := ch.Client()
			if err != nil {
				return err
			}

			if len(args) != 1 {
				return cmd.Usage()
			}

			token := args[0]

			req := &planetscale.DeleteServiceTokenRequest{
				ID:           token,
				Organization: ch.Config.Organization,
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting Token %s", printer.BoldBlue(token)))
			defer end()

			if err := client.ServiceTokens.Delete(ctx, req); err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("token does not exist in organization %s",
						printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Println("Token was successfully deleted.")
				return nil
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result": "token deleted",
				},
			)
		},
	}

	return cmd
}
