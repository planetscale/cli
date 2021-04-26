package token

import (
	"context"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func DeleteAccessCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-access <token> <access> <access> ...",
		Short: "delete access granted to a service token in the organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := ch.Client()
			if err != nil {
				return err
			}

			if len(args) < 2 {
				return cmd.Usage()
			}

			token, perms := args[0], args[1:]

			req := &planetscale.DeleteServiceTokenAccessRequest{
				ID:           token,
				Database:     ch.Config.Database,
				Organization: ch.Config.Organization,
				Accesses:     perms,
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Removing access %s on database %s", printer.BoldBlue(strings.Join(perms, ", ")), printer.BoldBlue(ch.Config.Database)))
			defer end()

			if err := client.ServiceTokens.DeleteAccess(ctx, req); err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s or token does not exist in organization %s\n",
						printer.BoldBlue(ch.Config.Database), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Accesses %v were successfully deleted!\n",
					printer.BoldBlue(strings.Join(perms, ",")))
				return nil
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result": "accesses deleted",
					"perms":  strings.Join(perms, ","),
				},
			)
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Database, "database", ch.Config.Database, "The database this project is using")
	cmd.MarkPersistentFlagRequired("database") // nolint:errcheck

	return cmd
}
