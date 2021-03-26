package backup

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// BackupCmd handles branch backups.
func BackupCmd(cfg *config.Config) *cobra.Command {
	createReq := &ps.CreateBackupRequest{}
	cmd := &cobra.Command{
		Use:     "backup <database> <branch> [options]",
		Short:   "Backup a branch's data and schema",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"b"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database := args[0]
			branch := args[1]

			createReq.Database = database
			createReq.Branch = branch
			createReq.Organization = cfg.Organization

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress(fmt.Sprintf("Creating backup of %s...", cmdutil.BoldBlue(branch)))
			defer end()
			backup, err := client.Backups.Create(ctx, createReq)
			if err != nil {
				return err
			}

			end()
			if cfg.OutputJSON {
				err := printer.PrintJSON(backup)
				if err != nil {
					return err
				}
			} else {
				fmt.Printf("Backup %s was successfully created!\n", cmdutil.BoldBlue(backup.Name))
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.Organization, "org", cfg.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(cfg))

	return cmd
}
