package router

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func DeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <database> <branch> <name>",
		Short:   "Delete a router from a Neki database branch",
		Args:    cmdutil.ExactArgs("database", "branch", "name"),
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, name := args[0], args[1], args[2]
			if !force {
				confirmationName := fmt.Sprintf("%s/%s/%s", database, branch, name)
				if err := ch.Printer.ConfirmCommand(confirmationName, "delete router", "deletion of router"); err != nil {
					return err
				}
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Deleting router %s from %s/%s", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			err = client.NekiRouters.Delete(cmd.Context(), &ps.DeleteNekiRouterRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Router:       name,
			})
			if err != nil {
				return handleError(err, database, branch, name)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Router %s was successfully deleted from %s/%s.\n", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result": "router deleted",
				"router": name,
				"branch": branch,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete the router without confirmation")
	return cmd
}
