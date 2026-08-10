package backup

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// PolicyListCmd lists backup policies for a database.
func PolicyListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database>",
		Short:   "List backup policies for a database",
		Args:    cmdutil.RequiredArgs("database"),
		Aliases: []string{"ls"},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching backup policies for %s", printer.BoldBlue(database)))
			defer end()

			policies, err := client.BackupPolicies.List(ctx, &ps.ListBackupPoliciesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(policies) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No backup policies exist for %s.\n", printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(toBackupPolicies(policies))
		},
	}

	return cmd
}
