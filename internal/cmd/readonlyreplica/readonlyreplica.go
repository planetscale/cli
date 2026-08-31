package readonlyreplica

import (
	"encoding/json"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// Cmd manages read-only replicas for Postgres branches.
func Cmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-only-replica <command>",
		Short: "Manage read-only replicas for a Postgres branch",
		Long: `Manage read-only replicas for a PostgreSQL database branch.

Read-only replicas provide dedicated capacity for queries that can tolerate
replication lag. They accept read traffic only.

This command is only available for PostgreSQL databases.`,
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(UpdateCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))

	return cmd
}

// ReadOnlyReplica is the human/JSON/CSV view of a Postgres read-only replica.
type ReadOnlyReplica struct {
	ID        string `header:"id" json:"id"`
	Name      string `header:"name" json:"name"`
	State     string `header:"state" json:"state"`
	Region    string `header:"region" json:"region"`
	Size      string `header:"size" json:"size"`
	Replicas  int    `header:"replicas" json:"replicas"`
	Ready     bool   `header:"ready" json:"ready"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`

	orig *ps.PostgresReadOnlyReplica
}

func (r *ReadOnlyReplica) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *ReadOnlyReplica) MarshalCSVValue() interface{} {
	return []*ReadOnlyReplica{r}
}

func toReadOnlyReplica(replica *ps.PostgresReadOnlyReplica) *ReadOnlyReplica {
	size := replica.ClusterDisplayName
	if size == "" {
		size = replica.ClusterName
	}
	if size == "" {
		size = "-"
	}

	region := replica.Region.Slug
	if region == "" {
		region = replica.Region.Name
	}
	if region == "" {
		region = "-"
	}

	return &ReadOnlyReplica{
		ID:        replica.ID,
		Name:      replica.Name,
		State:     replica.State,
		Region:    region,
		Size:      size,
		Replicas:  replica.Replicas,
		Ready:     replica.Ready,
		CreatedAt: printer.GetMilliseconds(replica.CreatedAt),
		orig:      replica,
	}
}

func toReadOnlyReplicas(replicas []*ps.PostgresReadOnlyReplica) []*ReadOnlyReplica {
	out := make([]*ReadOnlyReplica, 0, len(replicas))
	for _, replica := range replicas {
		out = append(out, toReadOnlyReplica(replica))
	}
	return out
}
