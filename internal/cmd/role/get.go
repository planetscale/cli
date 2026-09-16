package role

import (
	"fmt"
	"strings"

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
		router          string
		shard           string
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
				Router:          flags.router,
				Shard:           flags.shard,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					notFoundFormat, notFoundArgs := roleGetNotFound(roleID, branch, database, ch.Config.Organization, flags.readOnlyReplica, flags.bouncer, flags.router, flags.shard)

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

			if flags.bouncer != "" {
				bouncerURL := buildRoleConnectionURL(role.Username, role.Password, role.AccessHostURL, "6432", role.Options)
				if isNekiRole(role) {
					output := toNekiRole(role)
					output.DatabaseURL = bouncerURL
					return ch.Printer.PrintResource(output)
				}
				output := toPostgresRole(role)
				output.DatabaseURL = bouncerURL
				return ch.Printer.PrintResource(output)
			}

			return printRole(ch.Printer, role)
		},
	}

	cmd.Flags().BoolVar(&flags.replica, "replica", false, "Return connection details for a branch replica. On Neki this sets libpq options, not a username suffix.")
	cmd.Flags().StringVar(&flags.readOnlyReplica, "read-only-replica", "", "Return connection details for a read-only replica (name). Postgres only.")
	cmd.Flags().StringVar(&flags.bouncer, "bouncer", "", "Return connection details for a PgBouncer (name). Postgres only.")
	cmd.Flags().StringVar(&flags.router, "router", "", "Return connection details for a Neki router group (name). List routers with: pscale branch router list <database> <branch>.")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Return connection details that pin a Neki shard (name from pscale branch shard list).")
	cmd.MarkFlagsMutuallyExclusive("replica", "read-only-replica", "bouncer")
	cmd.MarkFlagsMutuallyExclusive("read-only-replica", "router")
	cmd.MarkFlagsMutuallyExclusive("read-only-replica", "shard")
	cmd.MarkFlagsMutuallyExclusive("bouncer", "router")
	cmd.MarkFlagsMutuallyExclusive("bouncer", "shard")

	return cmd
}

func roleGetNotFound(roleID, branch, database, org, readOnlyReplica, bouncer, router, shard string) (string, []any) {
	type extra struct {
		label string
		value string
	}

	extras := make([]extra, 0, 4)
	if readOnlyReplica != "" {
		extras = append(extras, extra{"read-only replica", readOnlyReplica})
	}
	if bouncer != "" {
		extras = append(extras, extra{"PgBouncer", bouncer})
	}
	if router != "" {
		extras = append(extras, extra{"router", router})
	}
	if shard != "" {
		extras = append(extras, extra{"shard", shard})
	}

	if len(extras) == 0 {
		return "role %s does not exist in branch %s of database %s (organization: %s)", []any{
			printer.BoldBlue(roleID),
			printer.BoldBlue(branch),
			printer.BoldBlue(database),
			printer.BoldBlue(org),
		}
	}

	parts := make([]string, 0, len(extras)+1)
	args := []any{printer.BoldBlue(roleID)}
	parts = append(parts, "role %s")
	for _, e := range extras {
		parts = append(parts, e.label+" %s")
		args = append(args, printer.BoldBlue(e.value))
	}

	return joinOr(parts) + " was not found in branch %s of database %s (organization: %s)", append(args,
		printer.BoldBlue(branch),
		printer.BoldBlue(database),
		printer.BoldBlue(org),
	)
}

func joinOr(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " or " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", or " + parts[len(parts)-1]
	}
}
