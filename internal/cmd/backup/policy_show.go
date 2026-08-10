package backup

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// PolicyShowCmd shows a single backup policy.
func PolicyShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <policy-id>",
		Short: "Show a backup policy",
		Args:  cmdutil.RequiredArgs("database", "policy-id"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			policyID := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching backup policy %s for %s", printer.BoldBlue(policyID), printer.BoldBlue(database)))
			defer end()

			policy, err := client.BackupPolicies.Get(ctx, &ps.GetBackupPolicyRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Policy:       policyID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("backup policy %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(policyID), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toBackupPolicy(policy))
		},
	}

	return cmd
}
