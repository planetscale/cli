package branch

import (
	"testing"
	"time"

	ps "github.com/planetscale/cli/internal/planetscale"

	qt "github.com/frankban/quicktest"
)

func TestBackupForRestorePoint(t *testing.T) {
	c := qt.New(t)

	restorePoint := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	older := &ps.Backup{
		PublicID:    "older",
		State:       "success",
		CompletedAt: restorePoint.Add(-2 * time.Hour),
	}
	selected := &ps.Backup{
		PublicID:    "selected",
		State:       "success",
		CompletedAt: restorePoint.Add(-1 * time.Hour),
	}
	tooLate := &ps.Backup{
		PublicID:    "too-late",
		State:       "success",
		CompletedAt: restorePoint.Add(1 * time.Hour),
	}

	backup, err := backupForRestorePoint([]*ps.Backup{older, selected, tooLate}, restorePoint)
	c.Assert(err, qt.IsNil)
	c.Assert(backup.PublicID, qt.Equals, "selected")

	_, err = backupForRestorePoint([]*ps.Backup{tooLate}, restorePoint)
	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Contains, "no successful backup found")
}
