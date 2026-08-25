package backup

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		protected bool
	}

	cmd := &cobra.Command{
		Use:   "update <database> <branch> <backup-id>",
		Short: "Update a backup's protected status",
		Long: `Update a backup's protected status.

--protected must be passed. Use --protected=false to disable protection.`,
		Example: `  pscale backup update mydb main backup-id --protected
  pscale backup update mydb main backup-id --protected=false`,
		Args: cmdutil.RequiredArgs("database", "branch", "backup-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, backup := args[0], args[1], args[2]

			if !cmd.Flags().Changed("protected") {
				return fmt.Errorf("--protected must be provided")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating backup %s for %s", printer.BoldBlue(backup), printer.BoldBlue(branch)))
			defer end()

			bkp, err := client.Backups.Update(ctx, &ps.UpdateBackupRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Backup:       backup,
				Protected:    flags.protected,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("backup %s does not exist in branch %s of %s (organization: %s)",
						printer.BoldBlue(backup), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			return ch.Printer.PrintResource(toBackup(bkp))
		},
	}

	cmd.Flags().BoolVar(&flags.protected, "protected", false,
		"Protect the backup from deletion (use --protected=false to disable)")

	return cmd
}
