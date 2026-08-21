package role

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

func GetCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		replica         bool
		readOnlyReplica string
		bouncer         string
	}

	cmd := &cobra.Command{
		Use:   "get <database> <branch> <role-id>",
		Short: "Retrieve information about a specific role",
		Args:  cmdutil.RequiredArgs("database", "branch", "role-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			roleID := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching role %s from %s/%s...", printer.BoldBlue(roleID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			role, err := client.PostgresRoles.Get(ctx, &ps.GetPostgresRoleRequest{
				Organization:    ch.Config.Organization,
				Database:        database,
				Branch:          branch,
				RoleId:          roleID,
				Replica:         flags.replica,
				ReadOnlyReplica: flags.readOnlyReplica,
				Bouncer:         flags.bouncer,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					notFoundFormat := "role %s does not exist in branch %s of database %s (organization: %s)"
					notFoundArgs := []any{
						printer.BoldBlue(roleID),
						printer.BoldBlue(branch),
						printer.BoldBlue(database),
						printer.BoldBlue(ch.Config.Organization),
					}
					if flags.readOnlyReplica != "" {
						notFoundFormat = "role %s or a read-only replica in region %s was not found in branch %s of database %s (organization: %s)"
						notFoundArgs = []any{
							printer.BoldBlue(roleID),
							printer.BoldBlue(flags.readOnlyReplica),
							printer.BoldBlue(branch),
							printer.BoldBlue(database),
							printer.BoldBlue(ch.Config.Organization),
						}
					} else if flags.bouncer != "" {
						notFoundFormat = "role %s or PgBouncer %s was not found in branch %s of database %s (organization: %s)"
						notFoundArgs = []any{
							printer.BoldBlue(roleID),
							printer.BoldBlue(flags.bouncer),
							printer.BoldBlue(branch),
							printer.BoldBlue(database),
							printer.BoldBlue(ch.Config.Organization),
						}
					}

					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err,
						"read_branch",
						notFoundFormat,
						notFoundArgs...)
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toPostgresRole(role))
		},
	}

	cmd.Flags().BoolVar(&flags.replica, "replica", false, "Return connection details for a branch replica.")
	cmd.Flags().StringVar(&flags.readOnlyReplica, "read-only-replica", "", "Return connection details for a regional read-only replica (region slug).")
	cmd.Flags().StringVar(&flags.bouncer, "bouncer", "", "Return connection details for a PgBouncer (name).")
	cmd.MarkFlagsMutuallyExclusive("replica", "read-only-replica", "bouncer")

	return cmd
}
