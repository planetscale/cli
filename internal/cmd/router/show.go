package router

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show <database> <branch> <name>",
		Short:   "Show a router for a Neki database branch",
		Args:    cmdutil.ExactArgs("database", "branch", "name"),
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, name := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching router %s", printer.BoldBlue(name)))
			defer end()

			router, err := client.NekiRouters.Get(cmd.Context(), &ps.GetNekiRouterRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Router:       name,
			})
			if err != nil {
				return handleError(err, database, branch, name)
			}

			parameters, err := client.NekiRouters.ListParameters(cmd.Context(), &ps.ListNekiRouterParametersRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Router:       name,
			})
			if err != nil {
				return handleError(err, database, branch, name)
			}
			end()

			if ch.Printer.Format() != printer.Human {
				return ch.Printer.PrintResource(toRouterDetail(router, parameters))
			}

			if err := ch.Printer.PrintResource(toRouter(router)); err != nil {
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
