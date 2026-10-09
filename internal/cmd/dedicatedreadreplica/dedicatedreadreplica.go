package dedicatedreadreplica

import (
	"encoding/json"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// Cmd manages dedicated read replicas for Postgres branches.
func Cmd(ch *cmdutil.Helper) *cobra.Command {
	return cmd(ch)
}

const deprecatedName = "read-only-replica"

const deprecationMessage = "use dedicated-read-replica instead"

// DeprecatedCmd preserves the previous command name without advertising it.
// Cobra only prints Deprecated for the command that runs, so subcommands warn
// through PersistentPreRunE the same way the deprecated workflow command does.
func DeprecatedCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := cmd(ch)
	cmd.Use = deprecatedName + " <command>"
	cmd.Hidden = true
	cmd.Deprecated = deprecationMessage
	cmd.PersistentPreRunE = cmdutil.WarnDeprecated(deprecatedName, deprecationMessage, cmdutil.CheckAuthentication(ch.Config))
	return cmd
}

func cmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dedicated-read-replica <command>",
		Short: "Manage dedicated read replicas for a Postgres branch",
		Long: `Manage dedicated read replicas for a PostgreSQL database branch.

Dedicated read replicas provide dedicated capacity for queries that can
tolerate replication lag. They accept read traffic only.

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

// DedicatedReadReplica is the human/JSON/CSV view of a Postgres dedicated read replica.
type DedicatedReadReplica struct {
	ID        string `header:"id" json:"id"`
	Name      string `header:"name" json:"name"`
	State     string `header:"state" json:"state"`
	Region    string `header:"region" json:"region"`
	Size      string `header:"size" json:"size"`
	Replicas  int    `header:"replicas" json:"replicas"`
	Ready     bool   `header:"ready" json:"ready"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`

	orig *ps.PostgresDedicatedReadReplica
}

func (r *DedicatedReadReplica) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *DedicatedReadReplica) MarshalCSVValue() interface{} {
	return []*DedicatedReadReplica{r}
}

func toDedicatedReadReplica(replica *ps.PostgresDedicatedReadReplica) *DedicatedReadReplica {
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

	return &DedicatedReadReplica{
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

func toDedicatedReadReplicas(replicas []*ps.PostgresDedicatedReadReplica) []*DedicatedReadReplica {
	out := make([]*DedicatedReadReplica, 0, len(replicas))
	for _, replica := range replicas {
		out = append(out, toDedicatedReadReplica(replica))
	}
	return out
}
