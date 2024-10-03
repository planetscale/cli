package keyspace

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ResizeStatusCmd(ch *cmdutil.Helper) *cobra.Command {
	statusReq := &ps.KeyspaceResizeStatusRequest{}

	cmd := &cobra.Command{
		Use:   "status <database> <branch> <keyspace>",
		Short: "Show the last resize operation on a branch's keyspace",
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			statusReq.Organization = ch.Config.Organization
			statusReq.Database = database
			statusReq.Branch = branch
			statusReq.Keyspace = keyspace

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Saving changes to keyspace %s in %s/%s", keyspace, database, branch))
			defer end()

			krr, err := client.Keyspaces.ResizeStatus(ctx, statusReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s, branch %s, or keyspace %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(keyspace), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toKeyspaceResizeRequest(krr))
		},
	}

	return cmd
}
