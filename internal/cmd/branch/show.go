package branch

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func ShowCmd(cfg *config.Config) *cobra.Command {
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
				fmt.Println("🌐  Redirecting you to your database branch in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s/branches/%s", cmdutil.ApplicationURL, cfg.Organization, source, branch))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Fetching branch %s for %s", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(source)))
			defer end()
			b, err := client.DatabaseBranches.Get(ctx, &planetscale.GetDatabaseBranchRequest{
				Organization: cfg.Organization,
				Database:     source,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						cmdutil.BoldBlue(branch), cmdutil.BoldBlue(source), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()
			err = printer.PrintOutput(cfg.OutputJSON, printer.NewDatabaseBranchPrinter(b))
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolP("web", "w", false, "Show a database branch in your web browser.")
	return cmd
}
