package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

func DemoteCmd(ch *cmdutil.Helper) *cobra.Command {
	demoteReq := &ps.DemoteRequest{}

	cmd := &cobra.Command{
		Use:   "demote <database> <branch> [options]",
		Short: "Demote a production branch to development",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			database := args[0]
			branch := args[1]

			demoteReq.Database = database
			demoteReq.Branch = branch
			demoteReq.Organization = ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Demoting %s branch in %s to development...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			demotionRequest, err := client.DatabaseBranches.Demote(ctx, demoteReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s",
						printer.BoldBlue(branch), printer.BoldBlue(database))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if demotionRequest == nil {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("%s branch was successfully demoted to development.\n", printer.BoldBlue(branch))
					return nil
				} else {
					dbBranch, err := client.DatabaseBranches.Get(cmd.Context(), &ps.GetDatabaseBranchRequest{
						Organization: ch.Config.Organization,
						Database:     database,
						Branch:       branch,
					})
					if err != nil {
						return cmdutil.HandleError(err)
					}
					return ch.Printer.PrintResource(ToDatabaseBranch(dbBranch))
				}
			}

			return nil
		},
	}

	return cmd
}
