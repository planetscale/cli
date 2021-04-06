package branch

import (
	"context"
	"fmt"
	"net/url"

	"github.com/pkg/browser"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CreateCmd(cfg *config.Config) *cobra.Command {
	createReq := &ps.CreateDatabaseBranchRequest{
		Branch: new(ps.DatabaseBranch),
	}

	cmd := &cobra.Command{
		Use:     "create <source-database> <branch> [options]",
		Short:   "Create a new branch from a database",
		Args:    cmdutil.RequiredArgs("source-database", "branch"),
		Aliases: []string{"b"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
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
				err := browser.OpenURL(fmt.Sprintf(
					"%s/%s/%s/branches?name=%s&notes=%s&showDialog=true",
					cmdutil.ApplicationURL, cfg.Organization, source, url.QueryEscape(createReq.Branch.Name), url.QueryEscape(createReq.Branch.Notes),
				))
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
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("%s does not exist in %s\n", cmdutil.BoldBlue(source), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()
			if cfg.OutputJSON {
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

	return cmd
}
