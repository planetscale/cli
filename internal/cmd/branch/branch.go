package branch

import (
	"encoding/json"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// BranchCmd handles the branching of a database.
func BranchCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "branch <command>",
		Short:             "Create, delete, diff, and manage branches",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(SwitchCmd(ch))
	cmd.AddCommand(DiffCmd(ch))
	cmd.AddCommand(SchemaCmd(ch))
	cmd.AddCommand(RefreshSchemaCmd(ch))
	cmd.AddCommand(PromoteCmd(ch))
	cmd.AddCommand(DemoteCmd(ch))
	cmd.AddCommand(VSchemaCmd(ch))
	cmd.AddCommand(KeyspaceCmd(ch))
	cmd.AddCommand(SafeMigrationsCmd(ch))

	return cmd
}

type DatabaseBranch struct {
	Name           string `header:"name" json:"name"`
	ParentBranch   string `header:"parent branch,n/a" json:"parent_branch"`
	Production     bool   `header:"production" json:"production"`
	SafeMigrations bool   `header:"safe migrations" json:"safe_migrations"`
	Ready          bool   `header:"ready" json:"ready"`
	CreatedAt      int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt      int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.DatabaseBranch
}

func (d *DatabaseBranch) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *DatabaseBranch) MarshalCSVValue() interface{} {
	return []*DatabaseBranch{d}
}

// Actor represents an actor for an action
type Actor struct {
	Name string `header:"name"`
}

type BranchDemotionRequest struct {
	ID        string `header:"id" json:"id"`
	Actor     *Actor `header:"actor" json:"actor"`
	State     string `header:"state" json:"state"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.BranchDemotionRequest
}

func ToBranchDemotionRequest(dr *ps.BranchDemotionRequest) *BranchDemotionRequest {
	newDR := &BranchDemotionRequest{
		ID:        dr.ID,
		State:     dr.State,
		CreatedAt: cmdutil.TimeToMilliseconds(dr.CreatedAt),
		UpdatedAt: cmdutil.TimeToMilliseconds(dr.UpdatedAt),
		orig:      dr,
	}

	if dr.Actor != nil {
		newDR.Actor = &Actor{
			Name: dr.Actor.Name,
		}
	}

	return newDR
}

func (d *BranchDemotionRequest) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *BranchDemotionRequest) MarshalCSVValue() interface{} {
	return []*BranchDemotionRequest{d}
}

// ToDatabaseBranch returns a struct that prints out the various fields of a
// database model.
func ToDatabaseBranch(db *ps.DatabaseBranch) *DatabaseBranch {
	return &DatabaseBranch{
		Name:           db.Name,
		ParentBranch:   db.ParentBranch,
		Production:     db.Production,
		SafeMigrations: db.SafeMigrations,
		Ready:          db.Ready,
		CreatedAt:      db.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt:      db.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		orig:           db,
	}
}

func toDatabaseBranches(branches []*ps.DatabaseBranch) []*DatabaseBranch {
	bs := make([]*DatabaseBranch, 0, len(branches))

	for _, db := range branches {
		bs = append(bs, ToDatabaseBranch(db))
	}

	return bs
}

type DatabaseBranchKeyspace struct {
	Name      string `header:"name" json:"name"`
	Shards    int    `header:"shards" json:"shards"`
	Sharded   bool   `header:"sharded" json:"sharded"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.Keyspace
}

func (d *DatabaseBranchKeyspace) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *DatabaseBranchKeyspace) MarshalCSVValue() interface{} {
	return []*DatabaseBranchKeyspace{d}
}

// ToDatabaseBranch returns a struct that prints out the various fields of a
// database model.
func ToDatabaseBranchKeyspace(ks *ps.Keyspace) *DatabaseBranchKeyspace {
	return &DatabaseBranchKeyspace{
		Name:      ks.Name,
		Shards:    ks.Shards,
		Sharded:   ks.Sharded,
		CreatedAt: ks.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt: ks.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		orig:      ks,
	}
}

func toDatabaseBranchkeyspaces(keyspaces []*ps.Keyspace) []*DatabaseBranchKeyspace {
	bs := make([]*DatabaseBranchKeyspace, 0, len(keyspaces))

	for _, ks := range keyspaces {
		bs = append(bs, ToDatabaseBranchKeyspace(ks))
	}

	return bs
}
