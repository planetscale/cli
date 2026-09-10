package role

import (
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		ttl             cmdutil.TTLFlag
		inheritedRoles  string
		withReplication bool
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch> <name>",
		Short: "Create a new role for a Postgres or Neki database branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "name"),
		Example: `  # Create a role with admin access
  pscale role create mydb main my-role --inherited-roles postgres

  # Create a role with REPLICATION privilege (requires the postgres inherited role)
  pscale role create mydb main replicator --inherited-roles postgres --with-replication`,
		RunE: func(cmd *cobra.Command, args []string) error {
			database := args[0]
			branch := args[1]
			name := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating role %s for %s/%s...", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			var inheritedRoles []string
			if flags.inheritedRoles != "" {
				inheritedRoles = strings.Split(flags.inheritedRoles, ",")
				// Trim whitespace from each role name
				for i := range inheritedRoles {
					inheritedRoles[i] = strings.TrimSpace(inheritedRoles[i])
				}
			}

			role, err := client.PostgresRoles.Create(cmd.Context(), &ps.CreatePostgresRoleRequest{
				Organization:    ch.Config.Organization,
				Database:        database,
				Branch:          branch,
				Name:            name,
				TTL:             int(flags.ttl.Value.Seconds()),
				InheritedRoles:  inheritedRoles,
				WithReplication: flags.withReplication,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						cmd.Context(), cmd, ch.Config, ch.Client, err,
						"create_branch_password or create_production_branch_password",
						"database %s or branch %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()
			if ch.Printer.Format() == printer.Human {
				saveWarning := printer.BoldRed("Please save the values below as they will not be shown again")
				ch.Printer.Printf("Role %s was successfully created in %s/%s.\n%s\n\n",
					printer.BoldBlue(role.Name), printer.BoldBlue(database), printer.BoldBlue(branch), saveWarning)
				printPostgresRoleCredentials(ch.Printer, toPostgresRole(role))
				return nil
			}

			return printRole(ch.Printer, role)
		},
	}
	cmd.PersistentFlags().Var(&flags.ttl, "ttl", `TTL defines the time to live for the role. Durations such as "30m", "24h", or bare integers such as "3600" (seconds) are accepted. The default TTL is 0s, which means the role will never expire.`)
	cmd.PersistentFlags().StringVar(&flags.inheritedRoles, "inherited-roles", "", "Comma-separated list of role names to inherit privileges from. Common values are 'pg_read_all_data' for read access, 'pg_write_all_data' for write access, and 'postgres' for admin access.")
	cmd.Flags().BoolVar(&flags.withReplication, "with-replication", false, "When enabled, the role is created with REPLICATION privilege for logical replication. Requires --inherited-roles to include 'postgres'.")

	return cmd
}
