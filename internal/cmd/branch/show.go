package branch

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <source-database> <branch>",
		Short: "Show a specific branch of a database",
		Args:  cmdutil.RequiredArgs("source-database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			source := args[0]
			branch := args[1]

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				ch.Printer.Println("🌐  Redirecting you to your database branch in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s/branches/%s", cmdutil.ApplicationURL, ch.Config.Organization, source, branch))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching branch %s for %s", printer.BoldBlue(branch), printer.BoldBlue(source)))
			defer end()
			b, err := client.DatabaseBranches.Get(ctx, &planetscale.GetDatabaseBranchRequest{
				Organization: ch.Config.Organization,
				Database:     source,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(source), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()

			return ch.Printer.PrintResource(toDatabaseBranch(b))
		},
	}

	cmd.Flags().BoolP("web", "w", false, "Show a database branch in your web browser.")
	return cmd
}
