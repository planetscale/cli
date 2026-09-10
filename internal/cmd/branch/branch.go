package branch

import (
	"encoding/json"
	"time"

	"github.com/planetscale/cli/internal/cmd/admin"
	"github.com/planetscale/cli/internal/cmd/branch/vtctld"
	"github.com/planetscale/cli/internal/cmd/configprofile"
	"github.com/planetscale/cli/internal/cmd/router"
	"github.com/planetscale/cli/internal/cmd/shard"
	"github.com/planetscale/cli/internal/cmd/sidecar"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

const (
	branchGroupDatabase      = "database"
	branchGroupMySQLPostgres = "mysql-postgres"
	branchGroupVitess        = "vitess"
	branchGroupPostgres      = "postgres"
	branchGroupPostgresNeki  = "postgres-neki"
	branchGroupNeki          = "neki"
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

	cmd.AddGroup(
		&cobra.Group{ID: branchGroupDatabase, Title: printer.Bold("MySQL, Postgres, and Neki branch commands:")},
		&cobra.Group{ID: branchGroupMySQLPostgres, Title: printer.Bold("MySQL and Postgres:")},
		&cobra.Group{ID: branchGroupVitess, Title: printer.Bold("Vitess/MySQL-specific:")},
		&cobra.Group{ID: branchGroupPostgres, Title: printer.Bold("Postgres-specific:")},
		&cobra.Group{ID: branchGroupPostgresNeki, Title: printer.Bold("Postgres and Neki:")},
		&cobra.Group{ID: branchGroupNeki, Title: printer.Bold("Neki-specific:")},
	)

	add := func(group string, command *cobra.Command) {
		command.GroupID = group
		cmd.AddCommand(command)
	}

	add(branchGroupDatabase, CreateCmd(ch))
	add(branchGroupDatabase, ListCmd(ch))
	add(branchGroupDatabase, DeleteCmd(ch))
	add(branchGroupDatabase, ShowCmd(ch))
	add(branchGroupDatabase, UpdateCmd(ch))
	add(branchGroupDatabase, SwitchCmd(ch))
	add(branchGroupDatabase, SchemaCmd(ch))
	add(branchGroupDatabase, PromoteCmd(ch))
	add(branchGroupDatabase, DemoteCmd(ch))
	add(branchGroupMySQLPostgres, ConnectionsCmd(ch))
	add(branchGroupDatabase, InfraCmd(ch))
	add(branchGroupVitess, DiffCmd(ch))
	add(branchGroupVitess, LintCmd(ch))
	add(branchGroupVitess, RefreshSchemaCmd(ch))
	add(branchGroupVitess, RoutingRulesCmd(ch))
	add(branchGroupVitess, SafeMigrationsCmd(ch))
	add(branchGroupVitess, QueryPatternsCmd(ch))
	add(branchGroupVitess, ProcesslistCmd(ch))
	add(branchGroupVitess, VtgateCmd(ch))
	add(branchGroupVitess, vtctld.VtctldCmd(ch))
	add(branchGroupPostgres, ResizeCmd(ch))
	add(branchGroupPostgres, ParametersCmd(ch))
	add(branchGroupPostgres, ExtensionsCmd(ch))
	add(branchGroupPostgres, SwitchoverCmd(ch))
	add(branchGroupPostgresNeki, MaintenanceCmd(ch))
	add(branchGroupNeki, DataTopologyCmd(ch))
	add(branchGroupNeki, admin.AdminCmd(ch))
	add(branchGroupNeki, configprofile.ConfigProfileCmd(ch))
	add(branchGroupNeki, router.RouterCmd(ch))
	add(branchGroupNeki, shard.ShardCmd(ch))
	add(branchGroupNeki, sidecar.SidecarCmd(ch))

	return cmd
}

type DatabaseBranch struct {
	ID             string `header:"id" json:"id"`
	Name           string `header:"name" json:"name"`
	ParentBranch   string `header:"parent branch,n/a" json:"parent_branch"`
	Region         string `header:"region" json:"region"`
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

// ToDatabaseBranch returns a struct that prints out the various fields of a
// database model.
func ToDatabaseBranch(db *ps.DatabaseBranch) *DatabaseBranch {
	return &DatabaseBranch{
		ID:             db.ID,
		Name:           db.Name,
		ParentBranch:   db.ParentBranch,
		Region:         db.Region.Slug,
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

type PostgresBranch struct {
	ID           string `header:"id" json:"id"`
	Name         string `header:"name" json:"name"`
	ParentBranch string `header:"parent branch,n/a" json:"parent_branch"`
	Region       string `header:"region" json:"region"`
	Production   bool   `header:"production" json:"production"`
	Ready        bool   `header:"ready" json:"ready"`
	CreatedAt    int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt    int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
	orig         *ps.PostgresBranch
}

func ToPostgresBranch(b *ps.PostgresBranch) *PostgresBranch {
	return &PostgresBranch{
		ID:           b.ID,
		Name:         b.Name,
		ParentBranch: b.ParentBranch,
		Region:       b.Region.Slug,
		Production:   b.Production,
		Ready:        b.Ready,
		CreatedAt:    b.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt:    b.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		orig:         b,
	}
}

func toPostgresBranches(branches []*ps.PostgresBranch) []*PostgresBranch {
	bs := make([]*PostgresBranch, 0, len(branches))

	for _, db := range branches {
		bs = append(bs, ToPostgresBranch(db))
	}

	return bs
}

func (p *PostgresBranch) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(p.orig, "", "  ")
}

func (p *PostgresBranch) MarshalCSVValue() interface{} {
	return []*PostgresBranch{p}
}
