package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// Process is the table/json representation of a single MySQL process.
type Process struct {
	ID      int64  `header:"id,text" json:"id"`
	User    string `header:"user" json:"user"`
	Host    string `header:"host" json:"host"`
	DB      string `header:"db" json:"db"`
	Command string `header:"command" json:"command"`
	Time    int64  `header:"time (seconds),text" json:"time"`
	State   string `header:"state" json:"state"`
	Info    string `header:"info" json:"info"`
}

func toProcesses(processes []ps.Process) []*Process {
	rows := make([]*Process, 0, len(processes))
	for _, p := range processes {
		rows = append(rows, &Process{
			ID:      p.ID,
			User:    p.User,
			Host:    p.Host,
			DB:      p.DB,
			Command: p.Command,
			Time:    p.Time,
			State:   p.State,
			Info:    p.Info,
		})
	}
	return rows
}

// ProcesslistCmd manages MySQL process lists for a Vitess branch.
func ProcesslistCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "processlist <command>",
		Short: "Show and kill running MySQL processes for a branch",
		Long: `Show and kill running MySQL processes for a branch.

This command is only supported for Vitess databases.`,
	}

	cmd.AddCommand(ProcesslistShowCmd(ch))
	cmd.AddCommand(ProcesslistKillCmd(ch))

	return cmd
}

// ProcesslistShowCmd shows the MySQL process list for a Vitess branch.
func ProcesslistShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
		shard    string
	}

	cmd := &cobra.Command{
		Use:   "show <database> <branch>",
		Short: "Show the running MySQL processes for a branch",
		Long: `Show the output of "SHOW FULL PROCESSLIST" for a branch.

This command is only supported for Vitess databases.

The process list is read from a single primary tablet. If the database has a
single unsharded keyspace, that primary is targeted automatically. If it has
multiple keyspaces, pass --keyspace; if the targeted keyspace is sharded, also
pass --shard. Process IDs shown here can be passed to
"pscale branch processlist kill".`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Fetching process list for %s\u2026",
					printer.BoldBlue(fmt.Sprintf("%s/%s/%s", ch.Config.Organization, database, branch))))
			defer end()

			result, err := client.Processlist.List(ctx, &ps.ProcesslistRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     flags.keyspace,
				Shard:        flags.shard,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err, "read_branch",
						"branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Process list for keyspace %s shard %s (tablet %s):\n",
					printer.BoldBlue(result.Keyspace), printer.BoldBlue(result.Shard), printer.BoldBlue(result.Tablet))
				return ch.Printer.PrintResource(toProcesses(result.Processes))
			}

			return ch.Printer.PrintJSON(result)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace to target (required when the database has multiple keyspaces)")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Shard to target (required when the targeted keyspace is sharded)")

	return cmd
}
