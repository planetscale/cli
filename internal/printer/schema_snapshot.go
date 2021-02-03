package printer

import (
	"github.com/planetscale/planetscale-go/planetscale"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

// SchemaSnapshot returns a table and json serializable schema snapshot.
type SchemaSnapshot struct {
	ID        string `header:"id" json:"id"`
	Name      string `header:"name" json:"name"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
}

// NewSchemaSnapshotPrinter returns a struct that prints out the various fields
// of a schema snapshot model.
func NewSchemaSnapshotPrinter(ss *planetscale.SchemaSnapshot) *SchemaSnapshot {
	return &SchemaSnapshot{
		ID:        ss.ID,
		Name:      ss.Name,
		CreatedAt: getMilliseconds(ss.CreatedAt),
		UpdatedAt: getMilliseconds(ss.UpdatedAt),
	}
}

func NewSchemaSnapshotSlicePrinter(schemaSnapshots []*ps.SchemaSnapshot) []*SchemaSnapshot {
	snapshots := make([]*SchemaSnapshot, 0, len(schemaSnapshots))

	for _, ss := range schemaSnapshots {
		snapshots = append(snapshots, NewSchemaSnapshotPrinter(ss))
	}

	return snapshots
}
