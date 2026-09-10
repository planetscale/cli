package sidecar

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List sidecars for a Neki database branch",
		Args:    cmdutil.ExactArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching sidecars for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			sidecars, err := client.NekiSidecars.List(cmd.Context(), &ps.ListNekiSidecarsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return handleError(err, database, branch, "")
			}
			end()

			if len(sidecars) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No sidecars found in %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(toSidecars(sidecars))
		},
	}

	return cmd
}
