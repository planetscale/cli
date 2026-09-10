package admin

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ParametersCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "parameters <database> <branch>",
		Short:   "List parameters for a Neki admin",
		Long:    "List parameters for a Neki admin.\n\nEach cluster has one admin.",
		Args:    cmdutil.ExactArgs("database", "branch"),
		Aliases: []string{"params"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching parameters for the admin of %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			parameters, err := client.NekiAdmins.ListParameters(cmd.Context(), &ps.ListNekiAdminParametersRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return handleError(err, database, branch)
			}
			end()

			if len(parameters) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No parameters found for the admin of %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(toParameters(parameters))
		},
	}

	return cmd
}
