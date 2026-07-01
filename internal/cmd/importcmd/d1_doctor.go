package importcmd

import (
	"github.com/spf13/cobra"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/import/d1"
)

func d1DoctorCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check prerequisites for D1 migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := d1.Doctor(cmd.Context())
			if err != nil {
				return writeD1(ch, d1.ErrorResponse("doctor", err))
			}
			return writeD1(ch, d1.DoctorResponse(result))
		},
	}

	return cmd
}
