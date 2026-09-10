package sidecar

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ParametersCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "parameters <database> <branch> <sidecar>",
		Short:   "List parameters for a Neki sidecar",
		Long:    "List parameters for a Neki sidecar.\n\n<sidecar> is the sidecar ID from `sidecar list`, or the configuration profile name.",
		Args:    cmdutil.ExactArgs("database", "branch", "sidecar"),
		Aliases: []string{"params"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, sidecar := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching parameters for sidecar %s", printer.BoldBlue(sidecar)))
			defer end()

			parameters, err := client.NekiSidecars.ListParameters(cmd.Context(), &ps.ListNekiSidecarParametersRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Sidecar:      sidecar,
			})
			if err != nil {
				return handleError(err, database, branch, sidecar)
			}
			end()

			if len(parameters) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No parameters found for sidecar %s.\n", printer.BoldBlue(sidecar))
				return nil
			}

			return ch.Printer.PrintResource(toParameters(parameters))
		},
	}

	return cmd
}
