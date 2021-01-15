package branch

import (
	"context"
	"fmt"
	"net/url"

	"github.com/pkg/browser"
	"github.com/planetscale/cli/cmdutil"
	"github.com/planetscale/cli/config"
	ps "github.com/planetscale/planetscale-go"
	"github.com/spf13/cobra"
)

// BranchCmd handles the branching of a database.
func BranchCmd(cfg *config.Config) *cobra.Command {
	createReq := &ps.CreateDatabaseBranchRequest{
		Branch: new(ps.DatabaseBranch),
	}

	cmd := &cobra.Command{
		Use:     "branch <source-database> <branch-name> [options]",
		Short:   "Branch a production database",
		Aliases: []string{"b"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			// If the user does not provide a source database and a branch name,
			// show the usage.
			if len(args) != 2 {
				return cmd.Usage()
			}

			source := args[0]
			branch := args[1]

			// Simplest case, the names are equivalent
			if source == branch {
				return fmt.Errorf("A branch named '%s' already exists", branch)
			}

			createReq.Branch.Name = branch

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to branch a database in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s/branches?name=%s&notes=%s&showDialog=true", cmdutil.ApplicationURL, cfg.Organization, source, url.QueryEscape(createReq.Branch.Name), url.QueryEscape(createReq.Branch.Notes)))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			dbBranch, err := client.DatabaseBranches.Create(ctx, cfg.Organization, source, createReq)
			if err != nil {
				return err
			}

			fmt.Printf("Successfully created database branch: %s\n", dbBranch.Name)

			return nil
		},
	}

	cmd.Flags().StringVar(&createReq.Branch.Notes, "notes", "", "notes for the database branch")
	cmd.Flags().BoolP("web", "w", false, "Create a branch in your web browser")
	cmd.AddCommand(ListCmd(cfg))
	cmd.AddCommand(StatusCmd(cfg))
	cmd.AddCommand(DeleteCmd(cfg))
	cmd.AddCommand(GetCmd(cfg))

	return cmd
}
