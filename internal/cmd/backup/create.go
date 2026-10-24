package backup

import (
	"fmt"
	"slices"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// emergencySources are the valid values for --emergency-source.
var emergencySources = []string{"primary", "replica"}

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &ps.CreateBackupRequest{}
	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Backup a branch's data and schema",
		Long: `Backup a branch's data and schema.

--emergency requests an emergency backup, which clones data from a live
tablet instead of waiting for the next scheduled backup. --emergency-source
selects which tablet type to clone from (primary or replica) and is required
with --emergency (and rejected otherwise). rdonly is not accepted.`,
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"b"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			if createReq.Emergency {
				if !cmd.Flags().Changed("emergency-source") {
					return fmt.Errorf("--emergency-source is required when --emergency is set")
				}
				if !slices.Contains(emergencySources, createReq.EmergencySource) {
					return fmt.Errorf("invalid --emergency-source %q, must be one of: %s", createReq.EmergencySource, strings.Join(emergencySources, ", "))
				}
			} else if cmd.Flags().Changed("emergency-source") {
				return fmt.Errorf("--emergency-source is only valid when --emergency is set")
			}

			createReq.Database = database
			createReq.Branch = branch
			createReq.Organization = ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating backup of %s...", printer.BoldBlue(branch)))
			defer end()

			bkp, err := client.Backups.Create(ctx, createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Backup %s was successfully created.\n", printer.BoldBlue(bkp.Name))
				return nil
			}

			return ch.Printer.PrintResource(toBackup(bkp))
		},
	}

	cmd.Flags().StringVar(&createReq.Name, "name", "", "Optional name for the backup")
	cmd.Flags().BoolVar(&createReq.Emergency, "emergency", false, "Request an emergency backup that clones data from a live tablet instead of waiting for the next scheduled backup. Requires --emergency-source.")
	cmd.Flags().StringVar(&createReq.EmergencySource, "emergency-source", "", "Tablet type to clone from for an emergency backup: primary or replica. Required with --emergency; rejected otherwise.")

	return cmd
}
