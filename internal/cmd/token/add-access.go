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

func AddAccessCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-access <token> <access> <access> ...",
		Short: "add access to a service token in the organization",
		Example: `The add-access command grants a service token specific access on a specific database.

For example, to give a service token the ability to create, read and delete branches on a specific database:

  pscale token add-access <token id> read_branch delete_branch create_branch --database <database name>

For a complete list of the access permissions that can be granted to a token, see: TODO.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			if len(args) < 2 {
				return cmd.Usage()
			}

			token, perms := args[0], args[1:]

			req := &planetscale.AddServiceTokenAccessRequest{
				ID:           token,
				Database:     ch.Config.Database,
				Organization: ch.Config.Organization,
				Accesses:     perms,
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Adding access %s to database %s", printer.BoldBlue(strings.Join(perms, ", ")), printer.BoldBlue(ch.Config.Database)))
			defer end()

			access, err := client.ServiceTokens.AddAccess(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s\n",
						printer.BoldBlue(ch.Config.Database), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()

			return ch.Printer.PrintResource(toServiceTokenAccesses(access))
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Database, "database", ch.Config.Database, "The database this project is using")
	cmd.MarkPersistentFlagRequired("database") // nolint:errcheck

	return cmd
}
