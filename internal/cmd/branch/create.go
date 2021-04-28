package branch

import (
	"context"
	"fmt"
	"net/url"

	"github.com/pkg/browser"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
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
			createReq.Organization = ch.Config.Organization

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				ch.Printer.Println("🌐  Redirecting you to branch a database in your web browser.")
				err := browser.OpenURL(fmt.Sprintf(
					"%s/%s/%s/branches?name=%s&notes=%s&showDialog=true",
					cmdutil.ApplicationURL, ch.Config.Organization, source, url.QueryEscape(createReq.Branch.Name), url.QueryEscape(createReq.Branch.Notes),
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
			dbBranch, err := client.DatabaseBranches.Create(ctx, createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("source database %s does not exist in organization %s\n",
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

	cmd.Flags().StringVar(&createReq.Branch.Notes, "notes", "", "notes for the database branch")
	cmd.Flags().StringVar(&createReq.Branch.ParentBranch, "from", "", "branch to be created from")
	cmd.Flags().BoolP("web", "w", false, "Create a branch in your web browser")

	return cmd
}
