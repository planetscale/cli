package backup

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
		Use:     "show <database> <branch> <backup>",
		Short:   "Show a specific backup of a branch",
		Aliases: []string{"get"},
		Args:    cmdutil.RequiredArgs("database", "branch", "backup"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database := args[0]
			branch := args[1]
			backup := args[2]

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to your backup in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s/%s/backups/%s", cmdutil.ApplicationURL, cfg.Organization, database, branch, backup))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Fetching backup %s for %s", cmdutil.BoldBlue(backup), cmdutil.BoldBlue(branch)))
			defer end()
			b, err := client.Backups.Get(ctx, &planetscale.GetBackupRequest{
				Organization: cfg.Organization,
				Database:     database,
				Branch:       branch,
				Backup:       backup,
			})
			if err != nil {
				return err
			}

			end()
			err = printer.PrintOutput(cfg.OutputJSON, printer.NewBackupPrinter(b))
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("backup %s does not exist in branch %s of %s (organization: %s)\n",
						cmdutil.BoldBlue(backup), cmdutil.BoldBlue(branch), cmdutil.BoldBlue(database), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolP("web", "w", false, "Show a branch backup in your web browser.")
	return cmd
}
