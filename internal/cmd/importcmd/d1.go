package importcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/import/d1"
	"github.com/planetscale/cli/internal/printer"
)

const defaultD1Branch = "main"

var d1DatabaseBranchArgs = cobra.RangeArgs(1, 2)

func parseDatabaseBranch(args []string) (database, branch string) {
	database = args[0]
	branch = defaultD1Branch
	if len(args) > 1 {
		branch = args[1]
	}
	return database, branch
}

func d1Org(ch *cmdutil.Helper) string {
	return ch.Config.Organization
}

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

func d1NotifyAPI(client *ps.Client, disabled bool) d1.NotifyAPIConfig {
	return d1.NotifyAPIConfig{Client: client, Disabled: disabled}
}

func importTableCount(prepared *d1.ImportPrepareResult) int {
	if prepared == nil || prepared.Plan == nil {
		return 0
	}
	return countDataTables(prepared.Plan.Tables)
}

func verifyTableCount(org, database, branch, migrationID, inputPath string) int {
	path := inputPath
	if path == "" && migrationID != "" {
		if state, err := d1.LoadState(org, database, branch, migrationID); err == nil {
			path = state.InputPath
		}
	}
	if path == "" {
		return 0
	}
	tables, err := d1.ParseDump(path)
	if err != nil {
		return 0
	}
	n := 0
	for _, t := range tables {
		if !d1.IsORMMetadataTable(t.Name) {
			n++
		}
	}
	return n
}

func countDataTables(tables []d1.TablePlan) int {
	n := 0
	for _, table := range tables {
		if !d1.IsORMMetadataTable(table.Name) {
			n++
		}
	}
	return n
}

// D1Cmd returns the import d1 subcommand group.
func D1Cmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "d1 <command>",
		Short: "Import Cloudflare D1 into PlanetScale Postgres",
		Long: `Offline import from Cloudflare D1 (SQLite) to PlanetScale Postgres.

Export your D1 database with wrangler (wrangler d1 export <name> --remote --output ./d1-export.sql),
lint the dump, then start the import (use --dry-run to preview).
All commands support --format json for machine-readable output.

Branch-scoped commands use the same positional form as other PlanetScale CLI commands:
  pscale import d1 start <database> [branch] --input ./d1-export.sql
Org comes from your pscale config (pscale org).`,
	}

	cmd.AddCommand(d1DoctorCmd(ch))
	cmd.AddCommand(d1LintCmd(ch))
	cmd.AddCommand(d1ConvertSchemaCmd(ch))
	cmd.AddCommand(d1StartCmd(ch))
	cmd.AddCommand(d1VerifyCmd(ch))
	cmd.AddCommand(d1StatusCmd(ch))
	cmd.AddCommand(d1CompleteCmd(ch))

	return cmd
}
