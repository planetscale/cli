package snapshot

import (
	"encoding/json"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// SnapshotCmd encapsulates the command for running snapshots.
func SnapshotCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "snapshot <action>",
		Short:             "Create, get, and list schema snapshots",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	return cmd
}

// SchemaSnapshot returns a table and json serializable schema snapshot.
type SchemaSnapshot struct {
	ID        string `header:"id" json:"id"`
	Name      string `header:"name" json:"name"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
	orig      *ps.SchemaSnapshot
}

func (s *SchemaSnapshot) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(s.orig, "", "  ")
}

func (s *SchemaSnapshot) MarshalCSVValue() interface{} {
	return []*SchemaSnapshot{s}
}

// toSchemaSnapshot returns a struct that prints out the various fields
// of a schema snapshot model.
func toSchemaSnapshot(ss *ps.SchemaSnapshot) *SchemaSnapshot {
	return &SchemaSnapshot{
		ID:        ss.ID,
		Name:      ss.Name,
		CreatedAt: printer.GetMilliseconds(ss.CreatedAt),
		UpdatedAt: printer.GetMilliseconds(ss.UpdatedAt),
		orig:      ss,
	}
}

func toSchemaSnapshots(schemaSnapshots []*ps.SchemaSnapshot) []*SchemaSnapshot {
	snapshots := make([]*SchemaSnapshot, 0, len(schemaSnapshots))

	for _, ss := range schemaSnapshots {
		snapshots = append(snapshots, toSchemaSnapshot(ss))
	}

	return snapshots
}
