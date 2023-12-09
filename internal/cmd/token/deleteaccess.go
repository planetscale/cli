package token

import (
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
		Example: `The delete-access command removes an access grant from a service token.

For example, to remove access for a specific database, include the --database flag:

  pscale service-token delete-access <token id> read_branch --database <database name>

To remove an organization level grant:
  
  pscale service-token add-access <token id> delete_databases`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
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
				Organization: ch.Config.Organization,
				Accesses:     perms,
			}

			if ch.Config.Database != "" {
				req.Database = ch.Config.Database
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Removing access %s on database %s", printer.BoldBlue(strings.Join(perms, ", ")), printer.BoldBlue(ch.Config.Database)))
			defer end()

			err = client.ServiceTokens.DeleteAccess(ctx, req)

			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("token %s does not exist in organization %s.\nPlease run 'pscale service-token list' to see a list of tokens",
						printer.BoldBlue(token), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				if len(perms) == 1 {
					ch.Printer.Printf("%v has been removed.\n",
						printer.BoldBlue(perms[0]))
				} else {
					ch.Printer.Printf("%v have been removed.\n",
						printer.BoldBlue(strings.Join(perms, ", ")))
				}
				return nil
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result": "access removed",
					"perms":  strings.Join(perms, ","),
				},
			)
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Database, "database", ch.Config.Database, "The database to remove access to")

	return cmd
}
