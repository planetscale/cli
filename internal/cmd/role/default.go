package role

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

func DefaultCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "default <database> <branch>",
		Short: "Show the default postgres role",
		Long: `Show the default postgres role for a branch.

This does not rotate credentials. Use 'pscale role reset-default' to issue a
new password for the default role.`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching default role for %s/%s...",
				printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			role, err := client.PostgresRoles.GetDefaultRole(ctx, &ps.GetDefaultPostgresRoleRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					// The default role endpoint also 404s when the branch exists
					// but its default credentials have never been created.
					if _, branchErr := client.PostgresBranches.Get(ctx, &ps.GetPostgresBranchRequest{
						Organization: ch.Config.Organization,
						Database:     database,
						Branch:       branch,
					}); branchErr == nil {
						end()
						return defaultRoleNotCreatedError(ch, database, branch)
					}

					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err,
						"read_branch",
						"database %s or branch %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
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

func defaultRoleNotCreatedError(ch *cmdutil.Helper, database, branch string) error {
	org := ch.Config.Organization
	if ch.Printer.Format() != printer.JSON {
		return fmt.Errorf("the default role for %s/%s has not been created yet. Run 'pscale role reset-default %s %s' to create it",
			printer.BoldBlue(database), printer.BoldBlue(branch), database, branch)
	}

	msg := fmt.Sprintf("the default role for %s/%s has not been created yet", database, branch)
	resp := cmdutil.JSONErrorResponse{
		Status: "action_required",
		Error:  msg,
		Issues: []cmdutil.JSONErrorIssue{{Code: "NOT_FOUND", Message: msg}},
		NextSteps: []string{
			"Ask the user to approve creating credentials for the default role",
			fmt.Sprintf("pscale role reset-default %s %s --org %s --format json --force", database, branch, org),
		},
	}
	if err := ch.Printer.PrintJSON(resp); err != nil {
		return err
	}
	return cmdutil.JSONReportedError(cmdutil.ActionRequestedExitCode)
}
