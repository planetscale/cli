package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

func DemoteCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "demote <database> <branch> [options]",
		Short: "Demote a production branch to development",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &ps.DemoteRequest{
				Database:     args[0],
				Branch:       args[1],
				Organization: ch.Config.Organization,
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Demoting %s branch in %s to development...",
				printer.BoldBlue(req.Branch), printer.BoldBlue(req.Database)))
			defer end()

			b, err := client.DatabaseBranches.Demote(cmd.Context(), req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s",
						printer.BoldBlue(req.Branch), printer.BoldBlue(req.Database))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("%s branch was successfully demoted to development.\n", printer.BoldBlue(req.Branch))
				return nil
			}

			return ch.Printer.PrintResource(ToDatabaseBranch(b))
		},
	}
}
