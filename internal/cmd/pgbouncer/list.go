package pgbouncer

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ListCmd lists dedicated PgBouncers for a branch.
func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List dedicated PgBouncers for a Postgres branch",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "PgBouncers"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching PgBouncers for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			bouncers, err := client.PostgresBouncers.List(ctx, &ps.ListPostgresBouncersRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s or branch %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(bouncers) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No dedicated PgBouncers exist for %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(toPgBouncers(bouncers))
		},
	}

	return cmd
}
