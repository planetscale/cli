package importcmd

import (
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/import/d1"
)

func d1VerifyCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		migrationID string
		input       string
		sqlite      string
		dbName      string
		noNotify    bool
	}

	cmd := &cobra.Command{
		Use:   "verify <database> [branch]",
		Short: "Verify D1 import (row counts, sequences, coercion, content checks)",
		Args:  d1DatabaseBranchArgs,
		Example: `  pscale import d1 verify mydb --migration-id abc123 --input ./d1-export.sql
  pscale import d1 verify mydb dev --migration-id abc123 --input ./d1-export.sql --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := parseDatabaseBranch(args)
			org := d1Org(ch)

			verifyOpts := d1.VerifyOptions{
				Org:         org,
				Database:    database,
				Branch:      branch,
				MigrationID: flags.migrationID,
				InputPath:   flags.input,
				SQLitePath:  flags.sqlite,
				DBName:      flags.dbName,
			}

			client, err := ch.Client()
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("verify", err))
			}
			verifyOpts.NotifyAPI = d1NotifyAPI(client, flags.noNotify)
			destURI, cleanup, err := d1.ResolveDestURI(cmd.Context(), client, d1.ImportOptions{
				Org:      org,
				Database: database,
				Branch:   branch,
				DBName:   flags.dbName,
			})
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("verify", err))
			}
			defer func() { _ = cleanup() }()
			verifyOpts.DestURI = destURI

			progress := newVerifyProgressReporter(ch, verifyTableCount(org, database, branch, flags.migrationID, flags.input))
			verifyOpts.OnProgress = progress.Report

			result, err := d1.Verify(cmd.Context(), verifyOpts)
			progress.Close()
			if err != nil {
				resp := d1.ErrorResponse("verify", err)
				if result != nil {
					resp.Data = result
				}
				return writeD1(ch, resp)
			}
			resp := d1.OKResponse("verify", result, d1.VerifyNextSteps(flags.migrationID, database, branch))
			resp.MigrationID = flags.migrationID
			resp.Phase = d1.PhaseVerified
			return writeD1(ch, resp)
		},
	}

	cmd.Flags().StringVar(&flags.migrationID, "migration-id", "", "Migration ID from plan/import")
	cmd.Flags().StringVar(&flags.input, "input", "", "Path to original D1 SQL export")
	cmd.Flags().StringVar(&flags.sqlite, "sqlite", "", "Path to local SQLite file for source counts")
	cmd.Flags().StringVar(&flags.dbName, "dbname", "postgres", "Destination PostgreSQL database name")
	cmd.Flags().BoolVar(&flags.noNotify, "no-notify", false, "Skip Slack notifications for this verification")
	cmd.MarkFlagRequired("migration-id")
	return cmd
}
