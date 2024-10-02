package keyspace

import (
	"encoding/json"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// KeyspaceCmd handles the management of keyspaces within a database branch.
func KeyspaceCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "keyspace <command>",
		Short:             "List, show, and manage keyspaces",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(VSchemaCmd(ch))
	cmd.AddCommand(CreateCmd(ch))

	return cmd
}

type BranchKeyspace struct {
	ID            string `json:"id"`
	Name          string `header:"name" json:"name"`
	Shards        int    `header:"shards" json:"shards"`
	Sharded       bool   `header:"sharded" json:"sharded"`
	Replicas      uint64 `header:"replicas" json:"replicas"`
	Resizing      bool   `header:"resizing" json:"resizing"`
	PendingResize bool   `header:"pending_resize" json:"resize_pending"`

	ClusterSize string `header:"cluster_size" json:"cluster_rate_name"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt   int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.Keyspace
}

func toBranchKeyspaces(keyspaces []*ps.Keyspace) []*BranchKeyspace {
	kss := make([]*BranchKeyspace, 0, len(keyspaces))

	for _, k := range keyspaces {
		kss = append(kss, toBranchKeyspace(k))
	}

	return kss
}

func toBranchKeyspace(k *ps.Keyspace) *BranchKeyspace {
	return &BranchKeyspace{
		ID:          k.ID,
		Name:        k.Name,
		Shards:      k.Shards,
		Sharded:     k.Sharded,
		Replicas:    k.Replicas,
		Resizing:    k.Resizing,
		ClusterSize: cmdutil.ToClusterSizeSlug(k.ClusterSize),
		CreatedAt:   cmdutil.TimeToMilliseconds(k.CreatedAt),
		UpdatedAt:   cmdutil.TimeToMilliseconds(k.UpdatedAt),
		orig:        k,
	}
}

func (k *BranchKeyspace) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(k.orig, "", "  ")
}

func (k *BranchKeyspace) MarshalCSVValue() interface{} {
	return []*BranchKeyspace{k}
}
