package importcmd

import (
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/import/d1"
)

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
