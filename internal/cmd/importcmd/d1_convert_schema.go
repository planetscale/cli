package importcmd

import (
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/import/d1"
)

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
