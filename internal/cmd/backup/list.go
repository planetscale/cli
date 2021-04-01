package backup

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ListCmd encapsulates the command for listing backups for a branch.
func ListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List all backups of a branch",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database := args[0]
			branch := args[1]

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to your backups in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s/%s/backups", cmdutil.ApplicationURL, cfg.Organization, database, branch))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Fetching backups for %s", cmdutil.BoldBlue(branch)))
			defer end()
			backups, err := client.Backups.List(ctx, &planetscale.ListBackupsRequest{
				Organization: cfg.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				if cmdutil.IsNotFoundError(err) {
					return fmt.Errorf("%s does not exist in %s\n", cmdutil.BoldBlue(branch), cmdutil.BoldBlue(database))
				}
				return errors.Wrap(err, "error listing backups")
			}
			end()

			if len(backups) == 0 && !cfg.OutputJSON {
				fmt.Printf("No backups exist in %s.\n", cmdutil.BoldBlue(branch))
				return nil
			}

			err = printer.PrintOutput(cfg.OutputJSON, printer.NewBackupSlicePrinter(backups))
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolP("web", "w", false, "List backups in your web browser.")
	return cmd
}
