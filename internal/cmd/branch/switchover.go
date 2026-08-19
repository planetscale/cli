package branch

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// SwitchoverCmd switches over the primary of a Postgres branch.
func SwitchoverCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		candidate string
	}

	cmd := &cobra.Command{
		Use:   "switchover <database> <branch>",
		Short: "Switch over the primary of a Postgres branch (Postgres only)",
		Long: `Switch over the primary of a Postgres branch. Postgres only.

On a branch with replicas the primary steps down and a replica is promoted in
its place; use --candidate to pick the replica, otherwise one is selected
automatically. A branch running a single instance has nothing to promote, so
that instance is restarted in place and the branch is unreachable while it
comes back. Writes are briefly interrupted while the switch completes.`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "Switchovers"); err != nil {
				return err
			}

			req := &ps.CreatePostgresSwitchoverRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Candidate:    flags.candidate,
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Switching over %s branch in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			switchover, err := client.PostgresSwitchovers.Create(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err, "write_database",
						"branch %s does not exist in database %s",
						printer.BoldBlue(branch), printer.BoldBlue(database))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Switchover %s for %s branch in %s was started.\n",
					printer.BoldBlue(switchover.ID), printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(ToPostgresSwitchover(switchover))
		},
	}

	cmd.Flags().StringVar(&flags.candidate, "candidate", "",
		"Name of the replica to promote, as returned by 'branch infra'. Omit to select automatically. Only applies to branches with replicas")

	return cmd
}

type PostgresSwitchover struct {
	ID     string `header:"id" json:"id"`
	State  string `header:"state" json:"state"`
	Method string `header:"method,n/a" json:"method,omitempty"`

	orig *ps.PostgresSwitchover
}

func (s *PostgresSwitchover) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(s.orig, "", "  ")
}

func (s *PostgresSwitchover) MarshalCSVValue() interface{} {
	return []*PostgresSwitchover{s}
}

func ToPostgresSwitchover(switchover *ps.PostgresSwitchover) *PostgresSwitchover {
	return &PostgresSwitchover{
		ID:     switchover.ID,
		State:  switchover.State,
		Method: switchover.Method,
		orig:   switchover,
	}
}
