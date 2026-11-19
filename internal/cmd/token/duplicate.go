package token

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func DuplicateCmd(ch *cmdutil.Helper) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "duplicate <id>",
		Short: "duplicate a service token in the organization",
		Args:  cmdutil.RequiredArgs("id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := ch.Client()
			if err != nil {
				return err
			}

			id := args[0]
			req := &planetscale.DuplicateServiceTokenRequest{
				Organization: ch.Config.Organization,
				ID:           id,
				Name:         name,
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Duplicating service token %s in org %s",
				printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization)))
			defer end()

			token, err := client.ServiceTokens.Duplicate(ctx, req)
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

			return ch.Printer.PrintResource(toServiceTokenWithSecret(token))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "name for the duplicated service token")
	cmd.MarkFlagRequired("name") // nolint:errcheck

	return cmd
}
