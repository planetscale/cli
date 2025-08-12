package role

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// ListCmd encapsulates the command for listing roles for a branch.
func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List all roles for a Postgres database branch",
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

			forMsg := fmt.Sprintf("%s/%s", printer.BoldBlue(database), printer.BoldBlue(branch))

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching roles for %s", forMsg))
			defer end()

			var allRoles []*ps.PostgresRole
			page := 1
			perPage := 100

			for {
				roles, err := client.PostgresRoles.List(ctx, &ps.ListPostgresRolesRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Branch:       branch,
				}, ps.WithPage(page), ps.WithPerPage(perPage))
				if err != nil {
					switch cmdutil.ErrCode(err) {
					case ps.ErrNotFound:
						return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
							printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
					default:
						return cmdutil.HandleError(err)
					}
				}

				allRoles = append(allRoles, roles...)

				// Check if there are more pages - if we got fewer results than perPage, we're done
				if len(roles) < perPage {
					break
				}
				page++
			}

			roles := allRoles
			end()

			if len(roles) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No roles exist in %s.\n", forMsg)
				return nil
			}

			return ch.Printer.PrintResource(toPostgresRoles(roles))
		},
	}

	return cmd
}

type PostgresRoleList struct {
	PublicID      string `header:"id" json:"id"`
	Name          string `header:"name" json:"name"`
	Username      string `header:"username" json:"username"`
	AccessHostURL string `header:"access_host_url" json:"access_host_url"`
	CreatedAt     string `header:"created_at" json:"created_at"`

	orig *ps.PostgresRole
}

func toPostgresRoles(roles []*ps.PostgresRole) []*PostgresRoleList {
	psRoles := make([]*PostgresRoleList, 0, len(roles))

	for _, role := range roles {
		psRoles = append(psRoles, &PostgresRoleList{
			PublicID:      role.ID,
			Name:          role.Name,
			Username:      role.Username,
			AccessHostURL: role.AccessHostURL,
			CreatedAt:     role.CreatedAt.Format("2006-01-02 15:04:05"),
			orig:          role,
		})
	}

	return psRoles
}