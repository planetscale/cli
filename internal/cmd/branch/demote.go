package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"

	ps "github.com/planetscale/cli/internal/planetscale"
)

func DemoteCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		clusterSize string
		replicas    int
	}

	cmd := &cobra.Command{
		Use:   "demote <database> <branch>",
		Short: "Demote a production branch to development",
		Long: `Demote a production branch to development.

This command is supported for MySQL databases and for enabled Postgres branches. Neki databases are not supported.

--cluster-size sets the Postgres cluster size to use. PS-DEV is allowed. Omit
it to leave the size unchanged. Changing the size requires the write_database
permission.

--replicas sets the Postgres replica count, not including the primary. 0 is a
single node. 2 to 8 is high availability. Omit it to leave the count
unchanged. Changing the replica count requires the write_database permission.

Vitess ignores both options.`,
		Example: `  pscale branch demote mydb main
  pscale branch demote mydb main --cluster-size PS-DEV
  pscale branch demote mydb main --cluster-size PS-10 --replicas 0`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &ps.DemoteRequest{
				Database:     args[0],
				Branch:       args[1],
				Organization: ch.Config.Organization,
				ClusterSize:  cmdutil.ToSizeSKUName(flags.clusterSize),
			}
			if cmd.Flags().Changed("replicas") {
				replicas := flags.replicas
				req.Replicas = &replicas
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Demoting %s branch in %s to development...",
				printer.BoldBlue(req.Branch), printer.BoldBlue(req.Database)))
			defer end()

			b, err := client.DatabaseBranches.Demote(cmd.Context(), req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						cmd.Context(), cmd, ch.Config, ch.Client, err, "connect_production_branch",
						"branch %s does not exist in database %s",
						printer.BoldBlue(req.Branch), printer.BoldBlue(req.Database))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("%s branch was successfully demoted to development.\n", printer.BoldBlue(req.Branch))
				return nil
			}

			return ch.Printer.PrintResource(ToDatabaseBranch(b))
		},
	}

	cmd.Flags().StringVar(&flags.clusterSize, "cluster-size", "", "Postgres cluster size to apply when demoting (for example, PS-DEV). Use 'pscale size cluster list' to see valid sizes.")
	cmd.Flags().IntVar(&flags.replicas, "replicas", 0, "Postgres replica count, not including the primary. 0 is a single node. 2 to 8 is high availability.")

	cmd.RegisterFlagCompletionFunc("cluster-size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.ClusterSizesIncludingDevelopmentCompletionFunc(ch, cmd, args, toComplete)
	})

	return cmd
}
