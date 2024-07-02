package keyspace

import (
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

	return cmd
}

type BranchKeyspace struct {
	ID            string `json:"id"`
	Name          string `header:"name" json:"name"`
	Shards        int    `header:"shards" json:"shards"`
	Sharded       bool   `header:"sharded" json:"sharded"`
	Replicas      uint64 `header:"replicas" json:"replicas"`
	ExtraReplicas uint64 `header:"extra_replicas" json:"extra_replicas"`
	Resizing      bool   `header:"resizing" json:"resizing"`
	ClusterSize   string `header:"cluster_size" json:"cluster_rate_name"`
	CreatedAt     int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt     int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

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
		ID:            k.ID,
		Name:          k.Name,
		Shards:        k.Shards,
		Sharded:       k.Sharded,
		Replicas:      k.Replicas,
		ExtraReplicas: k.ExtraReplicas,
		Resizing:      k.Resizing,
		ClusterSize:   cmdutil.ToClusterSizeSlug(k.ClusterSize),
		CreatedAt:     cmdutil.TimeToMilliseconds(k.CreatedAt),
		UpdatedAt:     cmdutil.TimeToMilliseconds(k.UpdatedAt),
		orig:          k,
	}
}
