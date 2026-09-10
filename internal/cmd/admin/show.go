package admin

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show <database> <branch>",
		Short:   "Show the admin for a Neki database branch",
		Long:    "Show the admin for a Neki database branch.\n\nEach cluster has one admin.",
		Args:    cmdutil.ExactArgs("database", "branch"),
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching admin for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			resource, err := client.NekiAdmins.Get(cmd.Context(), &ps.GetNekiAdminRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return handleError(err, database, branch)
			}

			parameters, err := client.NekiAdmins.ListParameters(cmd.Context(), &ps.ListNekiAdminParametersRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return handleError(err, database, branch)
			}
			end()

			if ch.Printer.Format() != printer.Human {
				return ch.Printer.PrintResource(toAdminDetail(resource, parameters))
			}

			if err := ch.Printer.PrintResource(toAdmin(resource)); err != nil {
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
