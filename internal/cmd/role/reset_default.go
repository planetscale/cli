package role

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ResetDefaultCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		force bool
	}

	cmd := &cobra.Command{
		Use:   "reset-default <database> <branch>",
		Short: "Reset the credentials for the default `postgres` role",
		Long:  "This command resets the credentials for the default `postgres` role in the database, allowing you to reconfigure access. Any connections using the `postgres` role will need to be updated with the new credentials.",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			org := ch.Config.Organization
			database := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Resetting default postgres role for %s/%s...", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			role, err := client.PostgresRoles.ResetDefaultRole(cmd.Context(), &ps.ResetDefaultRoleRequest{
				Organization: org,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				saveWarning := printer.BoldRed("Please save the values below as they will not be shown again")

				ch.Printer.Printf("Role was successfully reset for %s in %s.\n%s\n\n", printer.BoldBlue(branch), printer.BoldBlue(database), saveWarning)
			}

			return ch.Printer.PrintResource(toPostgresRole(role))
		},
	}

	cmd.Flags().BoolVar(&flags.force, "force", false, "Force reset without confirmation")

	return cmd
}

type PostgresRole struct {
	PublicID      string `header:"id" json:"id"`
	Name          string `header:"name" json:"name"`
	Username      string `header:"username" json:"username"`
	Password      string `header:"password" json:"password"`
	AccessHostURL string `header:"access_host_url" json:"access_host_url"`

	orig *ps.PostgresRole
}

func toPostgresRole(role *ps.PostgresRole) *PostgresRole {
	return &PostgresRole{
		PublicID:      role.ID,
		Name:          role.Name,
		Username:      role.Username,
		Password:      role.Password,
		AccessHostURL: role.AccessHostURL,
		orig:          role,
	}
}
