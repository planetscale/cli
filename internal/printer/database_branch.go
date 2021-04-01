package printer

import (
	"time"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type DatabaseBranch struct {
	Name         string `header:"name" json:"name"`
	Status       string `header:"status" json:"status"`
	ParentBranch string `header:"parent branch,n/a" json:"parent_branch"`
	CreatedAt    int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt    int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
	Notes        string `header:"notes" json:"notes"`
}

func NewDatabaseBranchPrinter(db *ps.DatabaseBranch) *ObjectPrinter {
	return &ObjectPrinter{
		Source:  db,
		Printer: newDatabaseBranchPrinter(db),
	}
}

// NewDatabaseBranchPrinter returns a struct that prints out the various fields of a
// database model.
func newDatabaseBranchPrinter(db *ps.DatabaseBranch) *DatabaseBranch {
	return &DatabaseBranch{
		Name:         db.Name,
		Notes:        db.Notes,
		Status:       db.Status,
		ParentBranch: db.ParentBranch,
		CreatedAt:    db.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt:    db.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
	}
}

func NewDatabaseBranchSlicePrinter(branches []*ps.DatabaseBranch) *ObjectPrinter {
	return &ObjectPrinter{
		Source:  branches,
		Printer: newDatabaseBranchSlicePrinter(branches),
	}
}

func newDatabaseBranchSlicePrinter(branches []*ps.DatabaseBranch) []*DatabaseBranch {
	bs := make([]*DatabaseBranch, 0, len(branches))

	for _, db := range branches {
		bs = append(bs, newDatabaseBranchPrinter(db))
	}

	return bs
}
