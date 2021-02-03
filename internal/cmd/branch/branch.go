package branch

import (
	"context"
	"fmt"
	"net/url"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
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

			createReq.Database = source
			createReq.Branch.Name = branch
			createReq.Organization = cfg.Organization

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

			end := cmdutil.PrintProgress(fmt.Sprintf("Creating branch from %s...", cmdutil.BoldBlue(source)))
			defer end()
			dbBranch, err := client.DatabaseBranches.Create(ctx, createReq)
			if err != nil {
				return err
			}

			isJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}

			end()
			if isJSON {
				err := printer.PrintJSON(dbBranch)
				if err != nil {
					return err
				}
			} else {
				fmt.Printf("Branch %s was successfully created!\n", cmdutil.BoldBlue(dbBranch.Name))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&createReq.Branch.Notes, "notes", "", "notes for the database branch")
	cmd.Flags().StringVar(&createReq.Branch.ParentBranch, "from", "", "branch to be created from")
	cmd.Flags().BoolP("web", "w", false, "Create a branch in your web browser")
	cmd.PersistentFlags().Bool("json", false, "Show output as JSON")

	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(cfg))
	cmd.AddCommand(StatusCmd(cfg))
	cmd.AddCommand(DeleteCmd(cfg))
	cmd.AddCommand(GetCmd(cfg))

	return cmd
}
