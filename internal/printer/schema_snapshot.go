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

func NewSchemaSnapshotPrinter(ss *planetscale.SchemaSnapshot) *ObjectPrinter {
	return &ObjectPrinter{
		Source:  ss,
		Printer: newSchemaSnapshotPrinter(ss),
	}
}

// newSchemaSnapshotPrinter returns a struct that prints out the various fields
// of a schema snapshot model.
func newSchemaSnapshotPrinter(ss *planetscale.SchemaSnapshot) *SchemaSnapshot {
	return &SchemaSnapshot{
		ID:        ss.ID,
		Name:      ss.Name,
		CreatedAt: getMilliseconds(ss.CreatedAt),
		UpdatedAt: getMilliseconds(ss.UpdatedAt),
	}
}

func NewSchemaSnapshotSlicePrinter(schemaSnapshots []*ps.SchemaSnapshot) *ObjectPrinter {
	return &ObjectPrinter{
		Source:  schemaSnapshots,
		Printer: newSchemaSnapshotSlicePrinter(schemaSnapshots),
	}
}

func newSchemaSnapshotSlicePrinter(schemaSnapshots []*ps.SchemaSnapshot) []*SchemaSnapshot {
	snapshots := make([]*SchemaSnapshot, 0, len(schemaSnapshots))

	for _, ss := range schemaSnapshots {
		snapshots = append(snapshots, newSchemaSnapshotPrinter(ss))
	}

	return snapshots
}
