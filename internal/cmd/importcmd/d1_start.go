package importcmd

import (
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/import/d1"
	"github.com/planetscale/cli/internal/printer"
)

func d1StartCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		input       string
		method      string
		migrationID string
		dbName      string
		dryRun      bool
		force       bool
		noNotify    bool
	}

	cmd := &cobra.Command{
		Use:   "start <database> [branch]",
		Short: "Start importing a D1 export (lint + plan, then load)",
		Long: `Runs lint and builds an import plan, then loads data into PlanetScale Postgres.
Requires pgloader on PATH — run import d1 doctor to verify prerequisites.

Use --dry-run to lint and save migration state without touching Postgres.`,
		Args: d1DatabaseBranchArgs,
		Example: `  # Preview lint + plan and get a migration ID
  pscale import d1 start mydb --input ./d1-export.sql --dry-run --format json

  # Run the import on a specific branch (human TTY prompts to confirm)
  pscale import d1 start mydb dev --input ./d1-export.sql --method pgloader --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := parseDatabaseBranch(args)
			org := d1Org(ch)

			importOpts := d1.ImportOptions{
				Org:         org,
				Database:    database,
				Branch:      branch,
				InputPath:   flags.input,
				Method:      flags.method,
				MigrationID: flags.migrationID,
				DBName:      flags.dbName,
				DryRun:      flags.dryRun,
			}

			prepared, err := d1.PrepareImport(importOpts)
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("start", err))
			}

			if !prepared.CanProceed {
				return writeD1(ch, d1.BlockedStartResponse(prepared, flags.dryRun))
			}

			if !flags.force && !flags.dryRun && ch.Printer.Format() == printer.Human {
				d1.PrintStartPreview(ch.Printer, prepared)
				if err := ch.Printer.ConfirmCommand(prepared.MigrationID, "import d1 start", "start"); err != nil {
					return err
				}
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}
			importOpts.NotifyAPI = d1NotifyAPI(client, flags.noNotify)

			var progress *progressReporter
			if !flags.dryRun {
				tableCount := importTableCount(prepared)
				progress = newImportProgressReporter(ch, tableCount, prepared.Plan.EstimatedSizeBytes)
				importOpts.OnProgress = progress.Report
				importOpts.PgloaderVerbose = ch.Debug()
			}

			result, err := d1.Import(cmd.Context(), client, &d1.DefaultImportClient{Client: client}, importOpts, prepared)
			if progress != nil {
				progress.Close()
			}
			if err != nil {
				resp := d1.ErrorResponse("start", err)
				if result != nil {
					resp.Data = result
					if result.Lint != nil {
						resp.Issues = result.Lint.Issues
					}
					resp.MigrationID = result.MigrationID
				} else {
					resp.MigrationID = prepared.MigrationID
				}
				return writeD1(ch, resp)
			}
			resp := d1.OKResponse("start", result, d1.StartNextSteps(result.MigrationID, database, branch, result.Method, flags.input, flags.dryRun))
			resp.MigrationID = result.MigrationID
			resp.Issues = result.Lint.Issues
			if flags.dryRun {
				resp.Status = "dry_run"
				resp.Phase = d1.PhasePlanned
			} else {
				resp.Phase = d1.PhaseImported
			}
			return writeD1(ch, resp)
		},
	}

	cmd.Flags().StringVar(&flags.input, "input", "", "Path to D1 SQL export")
	cmd.Flags().StringVar(&flags.method, "method", "", "Import method: pgloader (≥1GB) or psql (<1GB; schema via psql, data via pgloader)")
	cmd.Flags().StringVar(&flags.migrationID, "migration-id", "", "Existing migration ID from a prior start --dry-run")
	cmd.Flags().StringVar(&flags.dbName, "dbname", "postgres", "Destination PostgreSQL database name")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Lint and build import plan without loading Postgres")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&flags.noNotify, "no-notify", false, "Skip Slack notifications for this import")
	cmd.MarkFlagRequired("input")
	return cmd
}
