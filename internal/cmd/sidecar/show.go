package sidecar

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show <database> <branch> <sidecar>",
		Short:   "Show a sidecar for a Neki database branch",
		Long:    "Show a sidecar for a Neki database branch.\n\n<sidecar> is the sidecar ID from `sidecar list`, or the configuration profile name.",
		Args:    cmdutil.ExactArgs("database", "branch", "sidecar"),
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, sidecar := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching sidecar %s", printer.BoldBlue(sidecar)))
			defer end()

			resource, err := client.NekiSidecars.Get(cmd.Context(), &ps.GetNekiSidecarRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Sidecar:      sidecar,
			})
			if err != nil {
				return handleError(err, database, branch, sidecar)
			}

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

			if ch.Printer.Format() != printer.Human {
				return ch.Printer.PrintResource(toSidecarDetail(resource, parameters))
			}

			if err := ch.Printer.PrintResource(toSidecar(resource)); err != nil {
				return err
			}
			if len(parameters) == 0 {
				return nil
			}
			ch.Printer.Println()
			return ch.Printer.PrintResource(toParameters(parameters))
		},
	}

	return cmd
}
