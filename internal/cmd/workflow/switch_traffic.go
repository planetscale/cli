package workflow

import (
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

type switchTrafficFlags struct {
	replicasOnly bool
}

func SwitchTrafficCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags switchTrafficFlags

	cmd := &cobra.Command{
		Use:   "switch-traffic <database> <number>",
		Short: "Route queries to the target keyspace for a specific workflow in a PlanetScale database",
		Long: `Route queries to the target keyspace for a specific workflow in a PlanetScale database. 
By default, this command will route all queries for primary, replica, and read-only tablet). Use the --replicas-only flag to only route read queries from the replica and read-only tablets.`,
		Args: cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db, num := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			var number uint64
			number, err = strconv.ParseUint(num, 10, 64)
			if err != nil {
				return err
			}

			var workflow *ps.Workflow
			var end func()

			if flags.replicasOnly {
				end = ch.Printer.PrintProgress(fmt.Sprintf("Switching query traffic from replica and read-only tablets to the target keyspace for workflow %s in database %s…", printer.BoldBlue(number), printer.BoldBlue(db)))
				workflow, err = client.Workflows.SwitchReplicas(ctx, &ps.SwitchReplicasWorkflowRequest{
					Organization:   ch.Config.Organization,
					Database:       db,
					WorkflowNumber: number,
				})
			} else {
				end = ch.Printer.PrintProgress(fmt.Sprintf("Switching query traffic from primary, replica, and read-only tablets to the target keyspace for workflow %s in database %s…", printer.BoldBlue(number), printer.BoldBlue(db)))
				workflow, err = client.Workflows.SwitchPrimaries(ctx, &ps.SwitchPrimariesWorkflowRequest{
					Organization:   ch.Config.Organization,
					Database:       db,
					WorkflowNumber: number,
				})
			}
			defer end()
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s or workflow %s does not exist in organization %s",
						printer.BoldBlue(db), printer.BoldBlue(number), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				if flags.replicasOnly {
					ch.Printer.Printf("Successfully switched query traffic from replica and read-only tablets to target keyspace for workflow %s in database %s.\n",
						printer.BoldBlue(workflow.Name),
						printer.BoldBlue(db),
					)
					return nil
				}
				ch.Printer.Printf("Successfully switched queries from primary, replica, and read-only tablets to target keyspace for workflow %s in database %s.\n",
					printer.BoldBlue(workflow.Name),
					printer.BoldBlue(db),
				)
				return nil
			}

			return ch.Printer.PrintResource(toWorkflow(workflow))
		},
	}

	cmd.Flags().BoolVar(&flags.replicasOnly, "replicas-only", false, "Route read queries from the replica and read-only tablets to the target keyspace.")

	return cmd
}
