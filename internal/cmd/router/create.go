package router

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		size            string
		replicasPerCell int
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch> <name>",
		Short: "Create a router for a Neki database branch",
		Args:  cmdutil.ExactArgs("database", "branch", "name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, name := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating router %s in %s/%s", printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			request := &ps.CreateNekiRouterRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Name:         name,
				RouterSize:   cmdutil.ToSizeSKUName(flags.size),
			}
			if cmd.Flags().Changed("replicas-per-cell") {
				request.ReplicasPerCell = &flags.replicasPerCell
			}

			router, err := client.NekiRouters.Create(cmd.Context(), request)
			if err != nil {
				return handleError(err, database, branch, "")
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Router %s was successfully created in %s/%s.\n", printer.BoldBlue(router.Name), printer.BoldBlue(database), printer.BoldBlue(branch))
			}
			return ch.Printer.PrintResource(toRouter(router))
		},
	}

	cmd.Flags().StringVar(&flags.size, "size", "", "Router size SKU (e.g. NKR-5); the API picks a default if omitted")
	cmd.Flags().IntVar(&flags.replicasPerCell, "replicas-per-cell", 0, "Number of replicas in each cell")

	return cmd
}
