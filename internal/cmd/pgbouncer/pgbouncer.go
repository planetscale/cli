package pgbouncer

import (
	"encoding/json"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// Cmd manages dedicated PgBouncers for Postgres branches.
func Cmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pgbouncer <command>",
		Short: "Manage dedicated PgBouncers for a Postgres branch",
		Long: `Manage dedicated PgBouncers for a PostgreSQL database branch.

Dedicated PgBouncers run separately from the database nodes and provide
connection pooling for primary or replica traffic. This is distinct from the
local PgBouncer on port 6432 and from pgbouncer.* parameters on
'pscale branch resize'.

This command is only available for PostgreSQL databases.`,
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))

	return cmd
}

// PgBouncer is the human/JSON/CSV view of a dedicated PgBouncer.
type PgBouncer struct {
	ID              string `header:"id" json:"id"`
	Name            string `header:"name" json:"name"`
	Target          string `header:"target" json:"target"`
	Size            string `header:"size" json:"size"`
	ReplicasPerCell int    `header:"replicas_per_cell" json:"replicas_per_cell"`
	Actor           string `header:"actor" json:"actor"`
	CreatedAt       int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt       int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.PostgresBouncer
}

func (b *PgBouncer) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(b.orig, "", "  ")
}

func (b *PgBouncer) MarshalCSVValue() interface{} {
	return []*PgBouncer{b}
}

func toPgBouncer(bouncer *ps.PostgresBouncer) *PgBouncer {
	out := &PgBouncer{
		ID:              bouncer.ID,
		Name:            bouncer.Name,
		Target:          bouncer.Target,
		ReplicasPerCell: bouncer.ReplicasPerCell,
		Actor:           bouncer.Actor.Name,
		CreatedAt:       printer.GetMilliseconds(bouncer.CreatedAt),
		UpdatedAt:       printer.GetMilliseconds(bouncer.UpdatedAt),
		orig:            bouncer,
	}
	if bouncer.SKU != nil {
		if bouncer.SKU.DisplayName != "" {
			out.Size = bouncer.SKU.DisplayName
		} else {
			out.Size = bouncer.SKU.Name
		}
	} else if bouncer.BouncerSize != "" {
		out.Size = bouncer.BouncerSize
	} else {
		out.Size = "-"
	}
	if out.Actor == "" {
		out.Actor = "-"
	}
	return out
}

func toPgBouncers(bouncers []*ps.PostgresBouncer) []*PgBouncer {
	out := make([]*PgBouncer, 0, len(bouncers))
	for _, bouncer := range bouncers {
		out = append(out, toPgBouncer(bouncer))
	}
	return out
}
