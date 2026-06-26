package importcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/migrate/d1"
	"github.com/planetscale/cli/internal/printer"
)

func writeD1(ch *cmdutil.Helper, resp d1.Response) error {
	if resp.Status == "error" {
		switch ch.Printer.Format() {
		case printer.JSON:
			if err := ch.Printer.PrintJSON(resp); err != nil {
				return err
			}
		case printer.Human:
			d1.PrintHumanResponse(ch.Printer, resp)
		default:
			return fmt.Errorf(`import d1 does not support output format %q (use human or json)`, ch.Printer.Format())
		}
		return d1CommandError(resp)
	}

	switch ch.Printer.Format() {
	case printer.JSON:
		return ch.Printer.PrintJSON(resp)
	case printer.Human:
		d1.PrintHumanResponse(ch.Printer, resp)
		return nil
	default:
		return fmt.Errorf(`import d1 does not support output format %q (use human or json)`, ch.Printer.Format())
	}
}

func d1CommandError(resp d1.Response) error {
	msg := "import d1 command failed"
	if resp.Error != nil {
		msg = resp.Error.Message
		if resp.Error.Remediation != "" {
			msg += "\n" + resp.Error.Remediation
		}
	}
	return &cmdutil.Error{
		Msg:      msg,
		ExitCode: cmdutil.ActionRequestedExitCode,
		Printed:  true,
	}
}

// D1Cmd returns the import d1 subcommand group.
func D1Cmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "d1 <command>",
		Short: "Import Cloudflare D1 into PlanetScale Postgres",
		Long: `Offline import from Cloudflare D1 (SQLite) to PlanetScale Postgres.

Export with wrangler, lint the dump, then start the import (use --dry-run to preview).
All commands support --format json for machine-readable output.`,
	}

	cmd.AddCommand(d1DoctorCmd(ch))
	cmd.AddCommand(d1ExportCmd(ch))
	cmd.AddCommand(d1LintCmd(ch))
	cmd.AddCommand(d1ConvertSchemaCmd(ch))
	cmd.AddCommand(d1StartCmd(ch))
	cmd.AddCommand(d1VerifyCmd(ch))
	cmd.AddCommand(d1StatusCmd(ch))
	cmd.AddCommand(d1CompleteCmd(ch))

	return cmd
}

func d1DoctorCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check prerequisites for D1 migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := d1.Doctor(cmd.Context())
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("doctor", err))
			}
			if !result.Ready {
				return writeD1(ch, d1.ErrorResponse("doctor", d1.DoctorReadinessError(result)))
			}
			return writeD1(ch, d1.OKResponse("doctor", result, d1.DoctorNextSteps(result)))
		},
	}

	return cmd
}

func d1ExportCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		d1Database string
		output     string
		remote     bool
		table      string
		noData     bool
	}

	cmd := &cobra.Command{
		Use:     "export",
		Short:   "Export a D1 database using wrangler",
		Example: `  pscale import d1 export --d1-database my-app-db --remote --output ./d1-export.sql --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := d1.Export(cmd.Context(), d1.ExportOptions{
				D1Database: flags.d1Database,
				Output:     flags.output,
				Remote:     flags.remote,
				Table:      flags.table,
				NoData:     flags.noData,
			})
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("export", err))
			}
			resp := d1.OKResponse("export", result, []d1.NextStep{
				{Tool: "import_d1_lint", Command: "pscale import d1 lint --input " + result.OutputPath, Reason: "Analyze export before import"},
			})
			return writeD1(ch, resp)
		},
	}

	cmd.Flags().StringVar(&flags.d1Database, "d1-database", "", "Cloudflare D1 database name")
	cmd.Flags().StringVar(&flags.output, "output", "", "Output SQL file path")
	cmd.Flags().BoolVar(&flags.remote, "remote", false, "Export from remote D1 (not local dev)")
	cmd.Flags().StringVar(&flags.table, "table", "", "Export a single table")
	cmd.Flags().BoolVar(&flags.noData, "no-data", false, "Schema only export")
	cmd.MarkFlagRequired("d1-database")
	return cmd
}

func d1LintCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		input string
	}

	cmd := &cobra.Command{
		Use:     "lint",
		Short:   "Analyze a D1 SQL export for migration issues",
		Example: `  pscale import d1 lint --input ./d1-export.sql --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := d1.Lint(flags.input)
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("lint", err))
			}
			resp := d1.LintResponse(result)
			return writeD1(ch, resp)
		},
	}

	cmd.Flags().StringVar(&flags.input, "input", "", "Path to D1 SQL export")
	cmd.MarkFlagRequired("input")
	return cmd
}

func d1ConvertSchemaCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		input  string
		output string
	}

	cmd := &cobra.Command{
		Use:   "convert-schema",
		Short: "Convert SQLite schema in a D1 export to PostgreSQL DDL",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.output == "" {
				flags.output = flags.input + ".postgres.sql"
			}
			count, err := d1.ConvertSchema(flags.input, flags.output)
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("convert-schema", err))
			}
			resp := d1.OKResponse("convert-schema", map[string]any{
				"input":       flags.input,
				"output":      flags.output,
				"table_count": count,
			}, nil)
			return writeD1(ch, resp)
		},
	}

	cmd.Flags().StringVar(&flags.input, "input", "", "Path to D1 SQL export")
	cmd.Flags().StringVar(&flags.output, "output", "", "Output PostgreSQL schema file")
	cmd.MarkFlagRequired("input")
	return cmd
}

func d1StartCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		org         string
		database    string
		branch      string
		input       string
		method      string
		migrationID string
		dbName      string
		dryRun      bool
		force       bool
	}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start importing a D1 export (lint + plan, then load)",
		Long: `Runs lint and builds an import plan, then loads data into PlanetScale Postgres.
Requires pgloader on PATH — run import d1 doctor to verify prerequisites.

Use --dry-run to lint and save migration state without touching Postgres.`,
		Example: `  # Preview lint + plan and get a migration ID
  pscale import d1 start --org acme --database mydb --input ./d1-export.sql --dry-run --force --format json

  # Run the import
  pscale import d1 start --org acme --database mydb --input ./d1-export.sql --method pgloader --force --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			org := flags.org
			if org == "" {
				org = ch.Config.Organization
			}

			importOpts := d1.ImportOptions{
				Org:         org,
				Database:    flags.database,
				Branch:      flags.branch,
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

			result, err := d1.Import(cmd.Context(), client, &d1.DefaultImportClient{Client: client}, importOpts, prepared)
			if err != nil {
				resp := d1.ErrorResponse("start", err)
				if result != nil {
					resp.Data = result
					resp.Issues = result.Lint.Issues
				}
				resp.MigrationID = prepared.MigrationID
				return writeD1(ch, resp)
			}
			resp := d1.OKResponse("start", result, d1.StartNextSteps(result.MigrationID, flags.database, result.Method, flags.dryRun))
			resp.MigrationID = result.MigrationID
			resp.Issues = result.Lint.Issues
			if flags.dryRun {
				resp.Status = "dry_run"
			}
			return writeD1(ch, resp)
		},
	}

	cmd.Flags().StringVar(&flags.org, "org", "", "PlanetScale organization")
	cmd.Flags().StringVar(&flags.database, "database", "", "PlanetScale database name")
	cmd.Flags().StringVar(&flags.branch, "branch", "main", "PlanetScale branch name")
	cmd.Flags().StringVar(&flags.input, "input", "", "Path to D1 SQL export")
	cmd.Flags().StringVar(&flags.method, "method", "", "Import method: pgloader (≥1GB) or psql (<1GB; schema via psql, data via pgloader)")
	cmd.Flags().StringVar(&flags.migrationID, "migration-id", "", "Existing migration ID from a prior start --dry-run")
	cmd.Flags().StringVar(&flags.dbName, "dbname", "postgres", "Destination PostgreSQL database name")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Lint and build import plan without loading Postgres")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Skip confirmation prompt")
	cmd.MarkFlagRequired("database")
	cmd.MarkFlagRequired("input")
	return cmd
}

func d1VerifyCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		org         string
		database    string
		branch      string
		migrationID string
		input       string
		sqlite      string
	}

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify D1 import (row counts, sequences, coercion, content checks)",
		RunE: func(cmd *cobra.Command, args []string) error {
			org := flags.org
			if org == "" {
				org = ch.Config.Organization
			}

			verifyOpts := d1.VerifyOptions{
				Org:         org,
				Database:    flags.database,
				Branch:      flags.branch,
				MigrationID: flags.migrationID,
				InputPath:   flags.input,
				SQLitePath:  flags.sqlite,
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}
			destURI, cleanup, err := d1.ResolveDestURI(cmd.Context(), client, d1.ImportOptions{
				Org:      org,
				Database: flags.database,
				Branch:   flags.branch,
			})
			if err != nil {
				return err
			}
			defer func() { _ = cleanup() }()
			verifyOpts.DestURI = destURI

			result, err := d1.Verify(cmd.Context(), verifyOpts)
			if err != nil {
				resp := d1.ErrorResponse("verify", err)
				if result != nil {
					resp.Data = result
				}
				return writeD1(ch, resp)
			}
			resp := d1.OKResponse("verify", result, nil)
			resp.MigrationID = flags.migrationID
			return writeD1(ch, resp)
		},
	}

	cmd.Flags().StringVar(&flags.org, "org", "", "PlanetScale organization")
	cmd.Flags().StringVar(&flags.database, "database", "", "PlanetScale database name")
	cmd.Flags().StringVar(&flags.branch, "branch", "main", "PlanetScale branch name")
	cmd.Flags().StringVar(&flags.migrationID, "migration-id", "", "Migration ID from plan/import")
	cmd.Flags().StringVar(&flags.input, "input", "", "Path to original D1 SQL export")
	cmd.Flags().StringVar(&flags.sqlite, "sqlite", "", "Path to local SQLite file for source counts")
	cmd.MarkFlagRequired("database")
	return cmd
}

func d1StatusCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		org         string
		database    string
		branch      string
		migrationID string
	}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show local migration state",
		RunE: func(cmd *cobra.Command, args []string) error {
			org := flags.org
			if org == "" {
				org = ch.Config.Organization
			}
			state, err := d1.Status(org, flags.database, flags.branch, flags.migrationID)
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("status", err))
			}
			resp := d1.OKResponse("status", state, nil)
			resp.MigrationID = state.MigrationID
			return writeD1(ch, resp)
		},
	}

	cmd.Flags().StringVar(&flags.org, "org", "", "PlanetScale organization")
	cmd.Flags().StringVar(&flags.database, "database", "", "PlanetScale database name")
	cmd.Flags().StringVar(&flags.branch, "branch", "main", "PlanetScale branch name")
	cmd.Flags().StringVar(&flags.migrationID, "migration-id", "", "Migration ID")
	cmd.MarkFlagRequired("database")
	cmd.MarkFlagRequired("migration-id")
	return cmd
}

func d1CompleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		org         string
		database    string
		branch      string
		migrationID string
		force       bool
	}

	cmd := &cobra.Command{
		Use:     "complete",
		Aliases: []string{"teardown"},
		Short:   "Mark a D1 migration as complete in local state",
		RunE: func(cmd *cobra.Command, args []string) error {
			org := flags.org
			if org == "" {
				org = ch.Config.Organization
			}
			if !flags.force {
				if err := ch.Printer.ConfirmCommand(flags.migrationID, "import d1 complete", "complete"); err != nil {
					return err
				}
			}
			err := d1.Complete(org, flags.database, flags.branch, flags.migrationID)
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("complete", err))
			}
			return writeD1(ch, d1.OKResponse("complete", map[string]string{
				"migration_id": flags.migrationID,
				"status":       d1.PhaseComplete,
			}, nil))
		},
	}

	cmd.Flags().StringVar(&flags.org, "org", "", "PlanetScale organization")
	cmd.Flags().StringVar(&flags.database, "database", "", "PlanetScale database name")
	cmd.Flags().StringVar(&flags.branch, "branch", "main", "PlanetScale branch name")
	cmd.Flags().StringVar(&flags.migrationID, "migration-id", "", "Migration ID")
	cmd.Flags().BoolVar(&flags.force, "force", false, "Skip confirmation prompt")
	cmd.MarkFlagRequired("database")
	cmd.MarkFlagRequired("migration-id")
	return cmd
}
