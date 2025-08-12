package role

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name string
	}

	cmd := &cobra.Command{
		Use:   "update <database> <branch> <role-id>",
		Short: "Update a role's name",
		Args:  cmdutil.RequiredArgs("database", "branch", "role-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			roleID := args[2]

			if flags.name == "" {
				return fmt.Errorf("--name flag is required")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating role %s in %s/%s...", printer.BoldBlue(roleID), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			role, err := client.PostgresRoles.Update(ctx, &ps.UpdatePostgresRoleRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				RoleId:       roleID,
				Name:         flags.name,
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

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Role %s was successfully updated in %s/%s.\n",
					printer.BoldBlue(roleID), printer.BoldBlue(database), printer.BoldBlue(branch))
			}

			return ch.Printer.PrintResource(toPostgresRole(role))
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "New name for the role")
	cmd.MarkFlagRequired("name")

	return cmd
}