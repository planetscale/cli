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
	cmd.AddCommand(StatusCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(SwitchCmd(ch))
	cmd.AddCommand(DiffCmd(ch))
	cmd.AddCommand(SchemaCmd(ch))
	cmd.AddCommand(RefreshSchemaCmd(ch))
	cmd.AddCommand(PromoteCmd(ch))

	return cmd
}

type DatabaseBranch struct {
	Name         string `header:"name" json:"name"`
	Status       string `header:"status" json:"status"`
	ParentBranch string `header:"parent branch,n/a" json:"parent_branch"`
	Production   bool   `header:"production" json:"production"`
	CreatedAt    int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt    int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.DatabaseBranch
}

func (d *DatabaseBranch) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func (d *DatabaseBranch) MarshalCSVValue() interface{} {
	return []*DatabaseBranch{d}
}

// ToDatabaseBranch returns a struct that prints out the various fields of a
// database model.
func ToDatabaseBranch(db *ps.DatabaseBranch) *DatabaseBranch {
	return &DatabaseBranch{
		Name:         db.Name,
		Status:       db.Status,
		ParentBranch: db.ParentBranch,
		Production:   db.Production,
		CreatedAt:    db.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt:    db.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		orig:         db,
	}
}

func toDatabaseBranches(branches []*ps.DatabaseBranch) []*DatabaseBranch {
	bs := make([]*DatabaseBranch, 0, len(branches))

	for _, db := range branches {
		bs = append(bs, ToDatabaseBranch(db))
	}

	return bs
}

type DatabaseBranchStatus struct {
	Status string `header:"status" json:"status"`
}

func toDatabaseBranchStatus(status *ps.DatabaseBranchStatus) *DatabaseBranchStatus {
	var ready = "ready"
	if !status.Ready {
		ready = "not ready"
	}
	return &DatabaseBranchStatus{
		Status: ready,
	}
}
