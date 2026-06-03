package branch

import (
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// KillProcessResult is the CLI representation of a killed process response.
type KillProcessResult struct {
	Success  bool   `header:"success" json:"success"`
	Keyspace string `header:"keyspace" json:"keyspace"`
	Shard    string `header:"shard" json:"shard"`
	Tablet   string `header:"tablet" json:"tablet"`
	ID       int64  `header:"id,text" json:"id"`
	Kind     string `header:"kind" json:"kind"`
}

func (k *KillProcessResult) MarshalCSVValue() interface{} {
	return []*KillProcessResult{k}
}

func toKillProcessResult(result *ps.KillProcessResult) *KillProcessResult {
	return &KillProcessResult{
		Success:  result.Success,
		Keyspace: result.Keyspace,
		Shard:    result.Shard,
		Tablet:   result.Tablet,
		ID:       result.ID,
		Kind:     result.Kind,
	}
}

// ProcesslistKillCmd kills a running MySQL process on a Vitess branch.
func ProcesslistKillCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
		shard    string
		query    bool
	}

	cmd := &cobra.Command{
		Use:   "kill <database> <branch> <id>",
		Short: "Kill a running MySQL process on a branch (Vitess only)",
		Long: `Kill a MySQL process by ID, as shown in "pscale branch processlist show".

This command is only supported for Vitess (MySQL) databases.

By default the entire connection is killed (KILL <id>). Pass --query to kill
only the currently running statement (KILL QUERY <id>). The process must live on
the same primary tablet that the process list was read from, so the same
--keyspace/--shard targeting rules apply.`,
		Args: cmdutil.RequiredArgs("database", "branch", "id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			id, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil || id <= 0 {
				return fmt.Errorf("id must be a positive integer, got %q", args[2])
			}

			kind := "connection"
			if flags.query {
				kind = "query"
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Killing process %s on %s\u2026",
					printer.BoldBlue(id), printer.BoldBlue(fmt.Sprintf("%s/%s/%s", ch.Config.Organization, database, branch))))
			defer end()

			result, err := client.Processlist.Kill(ctx, &ps.KillProcessRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     flags.keyspace,
				Shard:        flags.shard,
				ID:           id,
				Kind:         kind,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err, "read_branch",
						"process %s or branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(id), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Killed %s %s on keyspace %s shard %s (tablet %s).\n",
					result.Kind, printer.BoldBlue(result.ID),
					printer.BoldBlue(result.Keyspace), printer.BoldBlue(result.Shard), printer.BoldBlue(result.Tablet))
				return nil
			}

			return ch.Printer.PrintResource(toKillProcessResult(result))
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace to target (required when the database has multiple keyspaces)")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Shard to target (required when the targeted keyspace is sharded)")
	cmd.Flags().BoolVar(&flags.query, "query", false, "Kill only the running query (KILL QUERY) instead of the whole connection")

	return cmd
}
