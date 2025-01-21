package keyspace

import (
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func RolloutStatusCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollout-status <database> <branch> <keyspace>",
		Short: "Show keyspace rollout status per shard",
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching rollout status for keyspace %s in %s/%s", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			k, err := client.Keyspaces.RolloutStatus(ctx, &ps.KeyspaceRolloutStatusRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     keyspace,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("keyspace %s does not exist in branch %s (database: %s, organization: %s)", printer.BoldBlue(keyspace), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toShardRollouts(k))
		},
	}

	return cmd
}

func toShardRollouts(keyspaceRollout *ps.KeyspaceRollout) []*ShardRollout {
	shards := make([]*ShardRollout, 0, len(keyspaceRollout.Shards))

	for _, s := range keyspaceRollout.Shards {
		shards = append(shards, toShardRollout(s))
	}

	return shards
}

type ShardRollout struct {
	Name  string `header:"name" json:"name"`
	State string `header:"state" json:"state"`

	LastRolloutStartedAt  *int64 `header:"started,timestamp(ms|utc|human)" json:"last_rollout_started_at"`
	LastRolloutFinishedAt *int64 `header:"finished,timestamp(ms|utc|human)" json:"last_rollout_finished_at"`

	orig *ps.ShardRollout
}

func toShardRollout(sr ps.ShardRollout) *ShardRollout {
	var startedAt, finishedAt *time.Time
	if !sr.LastRolloutStartedAt.IsZero() {
		startedAt = &sr.LastRolloutStartedAt
	}
	if !sr.LastRolloutFinishedAt.IsZero() {
		finishedAt = &sr.LastRolloutFinishedAt
	}
	return &ShardRollout{
		Name:                  sr.Name,
		State:                 cmdutil.SnakeToSentenceCase(sr.State),
		LastRolloutStartedAt:  printer.GetMillisecondsIfExists(startedAt),
		LastRolloutFinishedAt: printer.GetMillisecondsIfExists(finishedAt),
		orig:                  &sr,
	}
}
