package inspect

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/sqlquery"
)

// checkTimeout bounds a single diagnostic query so one slow check can't hang
// the command.
const checkTimeout = 30 * time.Second

type inspectFlags struct {
	keyspace   string
	postgresDB string
	role       string
	replica    bool
}

// InspectCmd runs read-only diagnostic checks against a database branch.
func InspectCmd(ch *cmdutil.Helper) *cobra.Command {
	flags := &inspectFlags{}

	cmd := &cobra.Command{
		Use:   "inspect <command>",
		Short: "Run read-only diagnostic checks against a database branch",
		Long: `Run read-only diagnostic checks against a database branch, using the same
ephemeral credentials as pscale sql. Each check is a single bounded query
against the engine's statistics tables. Nothing is written to the database.

Checks adapt to the database engine: MySQL (Vitess) checks read
information_schema, mysql, and sys; PostgreSQL checks read pg_catalog and
pg_stat views. Checks that don't apply to an engine explain what to use
instead.

On sharded Vitess databases, statistics reflect one shard's MySQL instance
per run. Pass --keyspace to pick the keyspace, or target an exact shard with
--keyspace 'mykeyspace/-80' (enumerate shards with SHOW VITESS_SHARDS via
pscale sql). Databases can have hundreds of shards, so no check fans out
across shards automatically.

On PostgreSQL, statistics are scoped to one database. Pass --dbname to target
the database your application uses (defaults to postgres).

For server-side, traffic-aware analysis (slow queries, schema recommendations,
anomalies), see pscale insights.`,
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck
	cmd.PersistentFlags().StringVar(&flags.keyspace, "keyspace", "",
		"Vitess keyspace to inspect, optionally with a shard and tablet type (e.g. mykeyspace, mykeyspace/-80, mykeyspace/-80@replica). List shards with: pscale sql <database> <branch> --query \"SHOW VITESS_SHARDS\". Defaults to @primary.")
	cmd.PersistentFlags().StringVar(&flags.postgresDB, "dbname", "postgres",
		"PostgreSQL database name to inspect")
	cmd.PersistentFlags().StringVar(&flags.role, "role", "",
		"Access role for the ephemeral credentials: reader, writer, readwriter, or admin. Defaults to reader. On PostgreSQL, the reader role may lack CONNECT on non-default databases; use --role admin if connecting with --dbname fails.")
	cmd.PersistentFlags().BoolVar(&flags.replica, "replica", false,
		"Run checks against a replica instead of the primary")

	for _, c := range checks {
		cmd.AddCommand(checkCmd(ch, c, flags))
	}
	cmd.AddCommand(allCmd(ch, flags))

	return cmd
}

// CheckResult is one check's outcome, printable in all output formats.
type CheckResult struct {
	Check    string           `json:"check"`
	Database string           `json:"database"`
	Branch   string           `json:"branch"`
	Columns  []string         `json:"columns,omitempty"`
	Rows     []map[string]any `json:"rows"`
	RowCount int              `json:"row_count"`
	Skipped  string           `json:"skipped,omitempty"`
	// NextSteps points at the pscale insights commands that analyze the same
	// problem from server-side production traffic data.
	NextSteps []string `json:"next_steps,omitempty"`
}

// Report is the combined output of inspect all.
type Report struct {
	Database string         `json:"database"`
	Branch   string         `json:"branch"`
	Results  []*CheckResult `json:"results"`
	// NextSteps recommends the server-side analysis commands that complement
	// these connection-level checks.
	NextSteps []string `json:"next_steps"`
}

func insightsNextSteps(organization, database, branch string) []string {
	return formatNextSteps([]string{
		"pscale insights queries <database> <branch>",
		"pscale insights errors <database> <branch>",
		"pscale insights anomalies <database> <branch>",
		"pscale insights recommendations <database>",
	}, organization, database, branch)
}

func formatNextSteps(steps []string, organization, database, branch string) []string {
	out := make([]string, 0, len(steps))
	for _, step := range steps {
		step = strings.ReplaceAll(step, "<database>", database)
		step = strings.ReplaceAll(step, "<branch>", branch)
		out = append(out, fmt.Sprintf("%s --org %s --format json", step, organization))
	}
	return out
}

func substituteResourceNames(s, database, branch string) string {
	s = strings.ReplaceAll(s, "<database>", database)
	return strings.ReplaceAll(s, "<branch>", branch)
}

func checkCmd(ch *cmdutil.Helper, c check, flags *inspectFlags) *cobra.Command {
	return &cobra.Command{
		Use:   c.Name + " <database> <branch>",
		Short: c.Short,
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			end := ch.Printer.PrintProgress(fmt.Sprintf("Inspecting %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			sess, err := newSession(ctx, ch, database, branch, flags)
			if err != nil {
				return cmdutil.HandleError(err)
			}
			defer sess.Close()

			result, err := runCheck(ctx, sess, c, ch.Config.Organization, database, branch)
			if err != nil {
				return cmdutil.HandleError(err)
			}
			end()

			if result.Skipped != "" && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("%s\n", result.Skipped)
				return nil
			}

			if err := printCheck(ch, c, result); err != nil {
				return err
			}
			if ch.Printer.Format() == printer.Human && len(result.NextSteps) > 0 {
				ch.Printer.Printf("\nFor server-side analysis of production traffic, also see:\n")
				for _, step := range result.NextSteps {
					ch.Printer.Printf("  %s\n", printer.BoldBlue(step))
				}
			}
			return nil
		},
	}
}

func allCmd(ch *cmdutil.Helper, flags *inspectFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "all <database> <branch>",
		Short: "Run every applicable check and print a combined report",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			end := ch.Printer.PrintProgress(fmt.Sprintf("Inspecting %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			sess, err := newSession(ctx, ch, database, branch, flags)
			if err != nil {
				return cmdutil.HandleError(err)
			}
			defer sess.Close()
			end()

			var results []*CheckResult
			for _, c := range checks {
				result, err := runCheck(ctx, sess, c, ch.Config.Organization, database, branch)
				if err != nil {
					// One failing check shouldn't abort the report.
					result = &CheckResult{
						Check:    c.Name,
						Database: database,
						Branch:   branch,
						Skipped:  err.Error(),
					}
				}
				results = append(results, result)

				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("\n%s — %s\n", printer.Bold(c.Name), c.Short)
					if result.Skipped != "" {
						ch.Printer.Printf("  %s\n", result.Skipped)
						continue
					}
					printHumanTable(ch, c, result)
				}
			}

			report := &Report{
				Database:  database,
				Branch:    branch,
				Results:   results,
				NextSteps: insightsNextSteps(ch.Config.Organization, database, branch),
			}

			switch ch.Printer.Format() {
			case printer.Human:
				ch.Printer.Printf("\nThese checks are point-in-time. For server-side analysis of production traffic\n(slow queries, failing queries, anomalies, schema recommendations), also run:\n")
				for _, step := range report.NextSteps {
					ch.Printer.Printf("  %s\n", printer.BoldBlue(step))
				}
				return nil
			case printer.JSON:
				return ch.Printer.PrintJSON(report)
			default:
				return fmt.Errorf("csv output is not supported for inspect all; use a single check or --format json")
			}
		},
	}
}

func newSession(ctx context.Context, ch *cmdutil.Helper, database, branch string, flags *inspectFlags) (*sqlquery.Session, error) {
	sess, err := sqlquery.NewSession(ctx, ch, sqlquery.Options{
		Organization:            ch.Config.Organization,
		Database:                database,
		Branch:                  branch,
		Keyspace:                flags.keyspace,
		PostgresDB:              flags.postgresDB,
		PostgresAdditionalRoles: []string{"pg_read_all_stats"},
		Role:                    flags.role,
		Replica:                 flags.replica,
	})
	if err != nil && strings.Contains(err.Error(), "permission denied for database") {
		return nil, fmt.Errorf("%w (the reader role may not have CONNECT on database %q; retry with --role admin)", err, flags.postgresDB)
	}
	return sess, err
}

func runCheck(ctx context.Context, sess *sqlquery.Session, c check, organization, database, branch string) (*CheckResult, error) {
	result := &CheckResult{Check: c.Name, Database: database, Branch: branch}

	var impl *engineSQL
	var hint string
	switch sess.Engine() {
	case "mysql":
		impl, hint = c.MySQL, c.MySQLHint
	default:
		impl, hint = c.Postgres, c.PostgresHint
	}
	if impl == nil {
		if hint == "" {
			hint = fmt.Sprintf("The %s check is not available for %s databases.", c.Name, sess.Engine())
		}
		result.Skipped = substituteResourceNames(hint, database, branch)
		result.NextSteps = formatNextSteps(c.NextSteps, organization, database, branch)
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	if impl.RequiresExtension != "" {
		// Extension names come from the check catalog above, never from user
		// input, so interpolating into the literal is safe.
		_, rows, err := sess.Query(ctx, fmt.Sprintf("SELECT 1 FROM pg_extension WHERE extname = '%s'", impl.RequiresExtension))
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			msg := fmt.Sprintf("The %s check needs the %q extension, which is not installed.", c.Name, impl.RequiresExtension)
			steps := formatNextSteps(c.NextSteps, organization, database, branch)
			if len(steps) > 0 {
				msg += fmt.Sprintf(" PlanetScale Insights provides this analysis server-side, no extension needed: %s.", strings.Join(steps, "; "))
			}
			msg += fmt.Sprintf(" To run this check anyway, enable the extension with: CREATE EXTENSION %s;", impl.RequiresExtension)
			result.Skipped = msg
			result.NextSteps = steps
			return result, nil
		}
	}

	columns, rows, err := sess.Query(ctx, impl.SQL)
	if err != nil {
		return nil, fmt.Errorf("%s check failed: %w", c.Name, err)
	}

	result.Columns = columns
	result.Rows = rows
	result.RowCount = len(rows)
	result.NextSteps = formatNextSteps(c.NextSteps, organization, database, branch)
	return result, nil
}

func printCheck(ch *cmdutil.Helper, c check, result *CheckResult) error {
	switch ch.Printer.Format() {
	case printer.JSON:
		return ch.Printer.PrintJSON(result)
	case printer.CSV:
		return printCSV(ch.Printer.ResourceOutput(), result)
	default:
		printHumanTable(ch, c, result)
		return nil
	}
}

func printHumanTable(ch *cmdutil.Helper, c check, result *CheckResult) {
	if len(result.Rows) == 0 {
		ch.Printer.Printf("  %s\n", c.EmptyMessage)
		return
	}

	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 2, 2, 2, ' ', 0)
	fmt.Fprintln(w, "  "+strings.Join(result.Columns, "\t"))
	for _, row := range result.Rows {
		values := make([]string, 0, len(result.Columns))
		for _, col := range result.Columns {
			values = append(values, formatValue(row[col]))
		}
		fmt.Fprintln(w, "  "+strings.Join(values, "\t"))
	}
	w.Flush()
	ch.Printer.Printf("%s", sb.String())
}

func printCSV(out io.Writer, result *CheckResult) error {
	w := csv.NewWriter(out)
	if result.Skipped != "" {
		if err := w.Write([]string{"check", "database", "branch", "skipped", "next_steps"}); err != nil {
			return err
		}
		if err := w.Write([]string{
			result.Check,
			result.Database,
			result.Branch,
			formatValue(result.Skipped),
			strings.Join(result.NextSteps, "; "),
		}); err != nil {
			return err
		}
	} else {
		if err := w.Write(result.Columns); err != nil {
			return err
		}
		for _, row := range result.Rows {
			values := make([]string, 0, len(result.Columns))
			for _, col := range result.Columns {
				values = append(values, formatValue(row[col]))
			}
			if err := w.Write(values); err != nil {
				return err
			}
		}
	}
	w.Flush()
	return w.Error()
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.Join(strings.Fields(s), " ")
	}
	return fmt.Sprintf("%v", v)
}
