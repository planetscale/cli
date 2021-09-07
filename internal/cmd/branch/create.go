package branch

import (
	"fmt"
	"net/url"

	"github.com/pkg/browser"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &ps.CreateDatabaseBranchRequest{}

	cmd := &cobra.Command{
		Use:     "create <source-database> <branch> [options]",
		Short:   "Create a new branch from a database",
		Args:    cmdutil.RequiredArgs("source-database", "branch"),
		Aliases: []string{"b"},
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
			source := args[0]
			branch := args[1]

			createReq.Database = source
			createReq.Name = branch
			createReq.Organization = ch.Config.Organization

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				ch.Printer.Println("🌐  Redirecting you to branch a database in your web browser.")
				err := browser.OpenURL(fmt.Sprintf(
					"%s/%s/%s/branches?name=%s&region=%s&showDialog=true",
					cmdutil.ApplicationURL, ch.Config.Organization, source, url.QueryEscape(createReq.Name), url.QueryEscape(createReq.Region),
				))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating branch from %s...", printer.BoldBlue(source)))
			defer end()
			dbBranch, err := client.DatabaseBranches.Create(cmd.Context(), createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("source database %s does not exist in organization %s",
						printer.BoldBlue(source), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Branch %s was successfully created.\n", printer.BoldBlue(dbBranch.Name))
				return nil
			}

			return ch.Printer.PrintResource(toDatabaseBranch(dbBranch))
		},
	}

	cmd.Flags().StringVar(&createReq.ParentBranch, "from", "", "branch to be created from")
	cmd.Flags().StringVar(&createReq.Region, "region", "", "region for the database")
	cmd.Flags().StringVar(&createReq.BackupID, "restore", "", "backup to restore into the branch")
	cmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx := cmd.Context()
		client, err := ch.Client()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		regions, err := client.Regions.List(ctx, &ps.ListRegionsRequest{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		regionStrs := make([]string, 0)

		for _, r := range regions {
			if r.Enabled {
				regionStrs = append(regionStrs, r.Slug)
			}
		}

		return regionStrs, cobra.ShellCompDirectiveDefault
	})
	cmd.Flags().BoolP("web", "w", false, "Create a branch in your web browser")

	return cmd
}
