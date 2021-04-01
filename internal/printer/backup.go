package printer

import (
	"time"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

type Backup struct {
	Name        string `header:"name" json:"name"`
	State       string `header:"state" json:"state"`
	Size        int64  `header:"size" json:"size"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt   int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
	StartedAt   int64  `header:"started_at,timestamp(ms|utc|human)" json:"started_at"`
	ExpiresAt   int64  `header:"expires_at,timestamp(ms|utc|human)" json:"expires_at"`
	CompletedAt int64  `header:"completed_at,timestamp(ms|utc|human)" json:"completed_at"`
}

func NewBackupPrinter(backup *ps.Backup) *ObjectPrinter {
	return &ObjectPrinter{
		Source:  backup,
		Printer: newBackupPrinter(backup),
	}
}

// newBackupPrinter Returns a struct that prints out the various fields of a branch model.
func newBackupPrinter(backup *ps.Backup) *Backup {
	return &Backup{
		Name:        backup.Name,
		State:       backup.State,
		Size:        backup.Size,
		CreatedAt:   backup.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt:   backup.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		StartedAt:   backup.StartedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		ExpiresAt:   backup.ExpiresAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		CompletedAt: backup.CompletedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
	}
}

func NewBackupSlicePrinter(backups []*ps.Backup) *ObjectPrinter {
	return &ObjectPrinter{
		Source:  backups,
		Printer: newBackupSlicePrinter(backups),
	}
}

func newBackupSlicePrinter(backups []*ps.Backup) []*Backup {
	bs := make([]*Backup, 0, len(backups))
	for _, backup := range backups {
		bs = append(bs, newBackupPrinter(backup))
	}
	return bs
}
