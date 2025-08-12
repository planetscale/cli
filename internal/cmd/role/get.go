package role

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func GetCmd(ch *cmdutil.Helper) *cobra.Command {
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
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				RoleId:       roleID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("role %s does not exist in branch %s of database %s (organization: %s)",
						printer.BoldBlue(roleID), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toPostgresRole(role))
		},
	}

	return cmd
}
