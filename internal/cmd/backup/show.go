package backup

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
		Use:   "show <database> <branch> <backup>",
		Short: "Show a specific backup of a branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "backup"),
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
				ch.Printer.Println("🌐  Redirecting you to your backup in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s/%s/%s/backups/%s", cmdutil.ApplicationURL, ch.Config.Organization, database, branch, backup))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching backup %s for %s", printer.BoldBlue(backup), printer.BoldBlue(branch)))
			defer end()
			bkp, err := client.Backups.Get(ctx, &planetscale.GetBackupRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Backup:       backup,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("backup %s does not exist in branch %s of %s (organization: %s)\n",
						printer.BoldBlue(backup), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()

			return ch.Printer.PrintResource(toBackups([]*planetscale.Backup{bkp}))
		},
	}

	cmd.Flags().BoolP("web", "w", false, "Show a branch backup in your web browser.")
	return cmd
}
