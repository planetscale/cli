package token

import (
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func AddAccessCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-access <token> <access> <access> ...",
		Short: "add access to a service token in the organization",
		Example: `The add-access command grants a service token access on a database or organization.

For example, to give a service token the ability to create, read and delete branches on a specific database:

  pscale service-token add-access <token id> read_branch delete_branch create_branch --database <database name>

To give a service token the ability to create and delete databases within the current organization:
  
  pscale service-token add-access <token id> delete_databases create_databases

For a complete list of the access permissions that can be granted to a token, see: https://api-docs.planetscale.com/reference/service-tokens#access-permissions.`,
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

			req := &planetscale.AddServiceTokenAccessRequest{
				ID:           token,
				Organization: ch.Config.Organization,
				Accesses:     perms,
			}

			if ch.Config.Database != "" {
				req.Database = ch.Config.Database
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Adding access %s to token %s",
				printer.BoldBlue(strings.Join(perms, ", ")), printer.BoldBlue(token)))
			defer end()

			access, err := client.ServiceTokens.AddAccess(ctx, req)
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

			return ch.Printer.PrintResource(toServiceTokenAccesses(access))
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Database, "database", ch.Config.Database, "The database to add access to")

	return cmd
}
