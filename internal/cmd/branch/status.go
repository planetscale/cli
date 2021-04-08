package branch

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// StatusCmd gets the status of a database branch using the PlanetScale API.
func StatusCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <database> <branch>",
		Short: "Check the status of a branch of a database",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			source := args[0]
			branch := args[1]

			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Getting status for branch %s in %s...", printer.BoldBlue(branch), printer.BoldBlue(source)))
			defer end()
			status, err := client.DatabaseBranches.GetStatus(ctx, &planetscale.GetDatabaseBranchStatusRequest{
				Organization: ch.Config.Organization,
				Database:     source,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(source), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}

			}

			end()

			return ch.Printer.PrintResource(toDatabaseBranchStatus(status))
		},
	}

	return cmd
}
