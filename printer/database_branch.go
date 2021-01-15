package printer

import (
	"time"

	ps "github.com/planetscale/planetscale-go"
)

type DatabaseBranch struct {
	Name         string `header:"name"`
	Notes        string `header:"notes"`
	Status       string `header:"status"`
	ParentBranch string `header:"parent branch,n/a"`
	CreatedAt    int64  `header:"created_at,timestamp(ms|utc|human)"`
	UpdatedAt    int64  `header:"updated_at,timestamp(ms|utc|human)"`
}

// NewDatabaseBranchPrinter returns a struct that prints out the various fields of a
// database model.
func NewDatabaseBranchPrinter(db *ps.DatabaseBranch) *DatabaseBranch {
	return &DatabaseBranch{
		Name:         db.Name,
		Notes:        db.Notes,
		Status:       db.Status,
		ParentBranch: db.ParentBranch,
		CreatedAt:    db.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt:    db.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
	}
}

func NewDatabaseBranchSlicePrinter(branches []*ps.DatabaseBranch) []*DatabaseBranch {
	bs := make([]*DatabaseBranch, 0, len(branches))

	for _, db := range branches {
		bs = append(bs, NewDatabaseBranchPrinter(db))
	}

	return bs
}
