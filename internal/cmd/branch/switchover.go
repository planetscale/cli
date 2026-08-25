package branch

import (
	"context"
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
	cmd.AddCommand(SwitchoverListCmd(ch), SwitchoverShowCmd(ch))

	return cmd
}

func SwitchoverListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List switchovers for a Postgres branch",
		Aliases: []string{"ls"},
		Args:    cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}
			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "Switchovers"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching switchovers for %s branch in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()
			switchovers, err := client.PostgresSwitchovers.List(ctx, &ps.ListPostgresSwitchoversRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			}, ps.WithPage(flags.page), ps.WithPerPage(flags.perPage))
			if err != nil {
				return switchoverReadError(ctx, cmd, ch, err, database, branch, "")
			}
			end()

			if len(switchovers) == 0 && ch.Printer.Format() == printer.Human {
				if flags.page > 0 {
					ch.Printer.Println("No switchovers found on this page.")
				} else {
					ch.Printer.Printf("No switchovers exist for %s branch in %s.\n", printer.BoldBlue(branch), printer.BoldBlue(database))
				}
				return nil
			}
			return ch.Printer.PrintResource(toPostgresSwitchovers(switchovers))
		},
	}
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")
	return cmd
}

func SwitchoverShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "show <database> <branch> <id>",
		Short: "Show a switchover for a Postgres branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, id := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}
			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "Switchovers"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching switchover %s for %s branch in %s...", printer.BoldBlue(id), printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()
			switchover, err := client.PostgresSwitchovers.Get(ctx, &ps.GetPostgresSwitchoverRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				ID:           id,
			})
			if err != nil {
				return switchoverReadError(ctx, cmd, ch, err, database, branch, id)
			}
			end()
			return ch.Printer.PrintResource(ToPostgresSwitchover(switchover))
		},
	}
}

func switchoverReadError(ctx context.Context, cmd *cobra.Command, ch *cmdutil.Helper, err error, database, branch, id string) error {
	if cmdutil.ErrCode(err) != ps.ErrNotFound {
		return cmdutil.HandleError(err)
	}
	if id != "" {
		return cmdutil.HandleNotFoundWithServiceTokenCheck(ctx, cmd, ch.Config, ch.Client, err, "read_branch",
			"switchover %s does not exist for branch %s in database %s",
			printer.BoldBlue(id), printer.BoldBlue(branch), printer.BoldBlue(database))
	}
	return cmdutil.HandleNotFoundWithServiceTokenCheck(ctx, cmd, ch.Config, ch.Client, err, "read_branch",
		"branch %s does not exist in database %s", printer.BoldBlue(branch), printer.BoldBlue(database))
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

func toPostgresSwitchovers(switchovers []*ps.PostgresSwitchover) []*PostgresSwitchover {
	result := make([]*PostgresSwitchover, 0, len(switchovers))
	for _, switchover := range switchovers {
		result = append(result, ToPostgresSwitchover(switchover))
	}
	return result
}
