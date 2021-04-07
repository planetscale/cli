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
		Use:   "branch <command>",
		Short: "Create, delete, and manage branches",
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

	return cmd
}

type DatabaseBranch struct {
	Name         string `header:"name" json:"name"`
	Status       string `header:"status" json:"status"`
	ParentBranch string `header:"parent branch,n/a" json:"parent_branch"`
	CreatedAt    int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt    int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
	Notes        string `header:"notes" json:"notes"`

	orig *ps.DatabaseBranch
}

func (d *DatabaseBranch) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

// toDatabaseBranch returns a struct that prints out the various fields of a
// database model.
func toDatabaseBranch(db *ps.DatabaseBranch) *DatabaseBranch {
	return &DatabaseBranch{
		Name:         db.Name,
		Notes:        db.Notes,
		Status:       db.Status,
		ParentBranch: db.ParentBranch,
		CreatedAt:    db.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt:    db.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		orig:         db,
	}
}

func toDatabaseBranches(branches []*ps.DatabaseBranch) []*DatabaseBranch {
	bs := make([]*DatabaseBranch, 0, len(branches))

	for _, db := range branches {
		bs = append(bs, toDatabaseBranch(db))
	}

	return bs
}

type DatabaseBranchStatus struct {
	Status      string `header:"status" json:"status"`
	GatewayHost string `header:"gateway_host" json:"gateway_host"`
	GatewayPort int    `header:"gateway_port,text" json:"gateway_port"`
	User        string `header:"username" json:"user"`
	Password    string `header:"password" json:"password"`

	orig *ps.DatabaseBranchStatus
}

func (d *DatabaseBranchStatus) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(d.orig, "", "  ")
}

func toDatabaseBranchStatus(status *ps.DatabaseBranchStatus) *DatabaseBranchStatus {
	var ready = "ready"
	if !status.Ready {
		ready = "not ready"
	}
	return &DatabaseBranchStatus{
		Status:      ready,
		GatewayHost: status.Credentials.GatewayHost,
		GatewayPort: status.Credentials.GatewayPort,
		User:        status.Credentials.User,
		Password:    status.Credentials.Password,
		orig:        status,
	}
}
