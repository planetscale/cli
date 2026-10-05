package branch

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ParametersCmd groups the read-only parameter commands for a Postgres
// branch. Parameters are changed with 'pscale branch resize --parameters'.
func ParametersCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		namespace string
		extension bool
		internal  bool
	}

	long := `List the configuration parameters of a Postgres branch, including their
current and default values. Values from a queued change request are reflected.

To change parameters, use 'pscale branch resize <database> <branch> --parameters namespace.name=value'.`

	run := func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		database, branch := args[0], args[1]

		client, err := ch.Client()
		if err != nil {
			return err
		}

		listReq := &ps.ListPostgresParametersRequest{
			Organization: ch.Config.Organization,
			Database:     database,
			Branch:       branch,
		}
		if cmd.Flags().Changed("extension") {
			listReq.Extension = &flags.extension
		}
		if cmd.Flags().Changed("internal") {
			listReq.Internal = &flags.internal
		}

		end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching parameters for branch %s in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
		defer end()

		parameters, err := client.PostgresBranches.ListParameters(ctx, listReq)
		if err != nil {
			switch cmdutil.ErrCode(err) {
			case ps.ErrNotFound:
				return fmt.Errorf("database %s or branch %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
			default:
				return cmdutil.HandleError(err)
			}
		}
		end()

		if flags.namespace != "" {
			filtered := parameters[:0]
			for _, param := range parameters {
				if param.Namespace == flags.namespace {
					filtered = append(filtered, param)
				}
			}
			parameters = filtered
		}

		return ch.Printer.PrintResource(toPostgresParameters(parameters))
	}

	registerFlags := func(cmd *cobra.Command) {
		cmd.Flags().StringVar(&flags.namespace, "namespace", "", "Only show parameters in this namespace (e.g. pgconf, pgbouncer, patroni).")
		cmd.Flags().BoolVar(&flags.extension, "extension", false, "Only show parameters that configure an extension (--extension=false hides them).")
		cmd.Flags().BoolVar(&flags.internal, "internal", false, "Only show internal parameters, which cannot be changed (--internal=false hides them).")
	}

	// The bare 'parameters <database> <branch>' invocation is an alias for
	// 'parameters list', so both commands share the flags and run function.
	cmd := &cobra.Command{
		Use:     "parameters <database> <branch>",
		Aliases: []string{"params"},
		Short:   "List the configuration parameters of a Postgres branch",
		Long:    long,
		Args:    cmdutil.RequiredArgs("database", "branch"),
		RunE:    run,
	}
	registerFlags(cmd)

	listCmd := &cobra.Command{
		Use:   "list <database> <branch>",
		Short: "List the configuration parameters of a Postgres branch",
		Long:  long,
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE:  run,
	}
	registerFlags(listCmd)
	cmd.AddCommand(listCmd)

	return cmd
}

type postgresParameter struct {
	Namespace string `header:"namespace" json:"namespace"`
	Name      string `header:"name" json:"name"`
	Value     string `header:"value" json:"value"`
	Default   string `header:"default" json:"default_value"`
	Type      string `header:"type" json:"parameter_type"`
	Restart   bool   `header:"restart" json:"restart"`

	orig *ps.PostgresParameter
}

func toPostgresParameters(parameters []*ps.PostgresParameter) []*postgresParameter {
	out := make([]*postgresParameter, 0, len(parameters))
	for _, param := range parameters {
		out = append(out, &postgresParameter{
			Namespace: param.Namespace,
			Name:      param.Name,
			Value:     formatParameterValue(param.Value),
			Default:   formatParameterValue(param.DefaultValue),
			Type:      param.ParameterType,
			Restart:   param.Restart,
			orig:      param,
		})
	}
	return out
}

// formatParameterValue renders a catalog value for display. The API documents
// values as strings, but the fields are typed any to survive numeric or
// boolean JSON; float64 needs explicit formatting or large values render in
// scientific notation (e.g. 1.073741824e+09).
func formatParameterValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (p *postgresParameter) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(p.orig, "", "  ")
}

func (p *postgresParameter) MarshalCSVValue() interface{} {
	return []*postgresParameter{p}
}
