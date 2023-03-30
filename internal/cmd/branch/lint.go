package branch

import (
	"errors"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// SchemaLintError returns a table-serializable lint error.
type SchemaLintError struct {
	LintError        string `json:"lint_error", header:"lint_error"`
	Keyspace         string `json:"keyspace_name", header:"keyspace"`
	Table            string `json:"table_name", header:"table"`
	SubjectType      string `json:"subject_type", header:"subject_type"`
	ErrorDescription string `json:"error_description", header:"error_description"`
	DocsURL          string `json:"docs_url", header:"docs_url"`

	orig *ps.SchemaLintError
}

func LintCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint <database> <branch>",
		Short: "Lints the schema for a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Linting schema for %s", printer.BoldBlue(branch)))
			defer end()

			lintErrors, err := client.DatabaseBranches.LintSchema(ctx, &ps.LintSchemaRequest{
				Organization: ch.Config.Organization,
				Database:     db,
				Branch:       branch,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()

			if ch.Printer.Format() == printer.Human {
				if len(lintErrors) > 0 {
					var sb strings.Builder
					sb.WriteString(printer.Red("Detected schema errors:\n\n"))
					for _, lintError := range lintErrors {
						fmt.Fprintf(&sb, "• %s\n", lintError.ErrorDescription)
					}

					return errors.New(sb.String())
				}

				ch.Printer.Println(printer.BoldGreen("No schema errors detected"))
				return nil
			} else {
				return ch.Printer.PrintResource(toSchemaLintErrors(lintErrors))
			}
		},
	}

	return cmd
}

// toSchemaLintError returns a struct that is serializable in multiple formats
func toSchemaLintError(err *ps.SchemaLintError) *SchemaLintError {
	return &SchemaLintError{
		LintError:        err.LintError,
		Keyspace:         err.Keyspace,
		Table:            err.Table,
		SubjectType:      err.SubjectType,
		ErrorDescription: err.ErrorDescription,
		DocsURL:          err.DocsURL,
		orig:             err,
	}
}

// toSchemaLintErrors returns a slice of serializable lint errors.
func toSchemaLintErrors(errs []*ps.SchemaLintError) []*SchemaLintError {
	lintErrors := make([]*SchemaLintError, len(errs))
	for i, err := range errs {
		lintErrors[i] = toSchemaLintError(err)
	}

	return lintErrors
}
