package pgbouncer

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ShowCmd shows a dedicated PgBouncer by name.
func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <branch> <name>",
		Short: "Show a dedicated PgBouncer",
		Args:  cmdutil.RequiredArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			name := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "PgBouncers"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching PgBouncer %s for %s/%s", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			bouncer, err := client.PostgresBouncers.Get(ctx, &ps.GetPostgresBouncerRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Bouncer:      name,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("PgBouncer %s does not exist on %s/%s (organization: %s)",
						printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toPgBouncer(bouncer))
		},
	}

	return cmd
}
