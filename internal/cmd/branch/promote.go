package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

func PromoteCmd(ch *cmdutil.Helper) *cobra.Command {
	promoteReq := &ps.PromoteRequest{}

	cmd := &cobra.Command{
		Use:   "promote <database> <branch> [options]",
		Short: "Promote a new branch from a database",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			client, err := ch.Client()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			org := ch.Config.Organization // --org flag
			if org == "" {
				cfg, err := ch.ConfigFS.DefaultConfig()
				if err != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}

				org = cfg.Organization
			}

			databases, err := client.Databases.List(cmd.Context(), &ps.ListDatabasesRequest{
				Organization: org,
			})
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			candidates := make([]string, 0, len(databases))
			for _, db := range databases {
				candidates = append(candidates, db.Name)
			}

			return candidates, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			db := args[0]
			branch := args[1]

			promoteReq.Database = db
			promoteReq.Organization = ch.Config.Organization
			promoteReq.Branch = branch

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Promoting %s branch in %s to production...", printer.BoldBlue(branch), printer.BoldBlue(db)))
			defer end()
			dbBranch, err := client.DatabaseBranches.Promote(cmd.Context(), promoteReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						cmd.Context(), cmd, ch.Config, ch.Client, err, "connect_production_branch",
						"branch %s does not exist in database %s",
						printer.BoldBlue(branch), printer.BoldBlue(db))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Branch %s in %s was successfully promoted to production. Note: In order to use deploy requests to deploy schema changes to this branch, safe migrations must also be enabled.\n", printer.BoldBlue(branch), printer.BoldBlue(db))
				return nil
			}

			return ch.Printer.PrintResource(ToDatabaseBranch(dbBranch))
		},
	}

	return cmd
}
