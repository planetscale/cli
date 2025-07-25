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
		Long:              "List, show, and manage keyspaces.\n\nThis command is only supported for Vitess databases.",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(VSchemaCmd(ch))
	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(ResizeCmd(ch))
	cmd.AddCommand(RolloutStatusCmd(ch))
	cmd.AddCommand(UpdateSettingsCmd(ch))
	cmd.AddCommand(SettingsCmd(ch))

	return cmd
}

type Keyspace struct {
	ID            string `json:"id"`
	Name          string `header:"name" json:"name"`
	Shards        int    `header:"shards" json:"shards"`
	Sharded       bool   `header:"sharded" json:"sharded"`
	Replicas      uint64 `header:"replicas" json:"replicas"`
	ExtraReplicas uint64 `header:"extra_replicas" json:"extra_replicas"`
	Resizing      bool   `header:"resizing" json:"resizing"`
	PendingResize bool   `header:"pending_resize" json:"resize_pending"`

	ClusterSize string `header:"cluster_size" json:"cluster_name"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt   int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.Keyspace
}

type KeyspaceSettings struct {
	ReplicationDurabilityConstraintStrategy string            `header:"replication durability constraint strategy" json:"replication_durability_constraint"`
	VReplicationFlags                       VReplicationFlags `header:"inline" json:"vreplication_flags"`

	orig *ps.Keyspace
}

func (k *KeyspaceSettings) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(k.orig, "", "  ")
}

func (k *KeyspaceSettings) MarshalCSVValue() interface{} {
	return []*KeyspaceSettings{k}
}

type VReplicationFlags struct {
	OptimizeInserts           bool `header:"optimize inserts" json:"optimize_inserts"`
	AllowNoBlobBinlogRowImage bool `header:"no blob binlog row image" json:"allow_no_blob_binlog_row_image"`
	VPlayerBatching           bool `header:"vplayer batching" json:"vplayer_batching"`
}

func toKeyspaces(keyspaces []*ps.Keyspace) []*Keyspace {
	kss := make([]*Keyspace, 0, len(keyspaces))

	for _, k := range keyspaces {
		kss = append(kss, toKeyspace(k))
	}

	return kss
}

func toKeyspace(k *ps.Keyspace) *Keyspace {
	return &Keyspace{
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

func (k *Keyspace) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(k.orig, "", "  ")
}

func (k *Keyspace) MarshalCSVValue() interface{} {
	return []*Keyspace{k}
}
