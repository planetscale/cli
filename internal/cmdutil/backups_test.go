package cmdutil

import (
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	ps "github.com/planetscale/cli/internal/planetscale"
)

func TestBackupForRestorePoint(t *testing.T) {
	c := qt.New(t)

	restorePoint := time.Date(2026, 8, 24, 12, 14, 49, 0, time.UTC)
	backups := []*ps.Backup{
		{
			PublicID:    "too-new",
			State:       "success",
			CompletedAt: time.Date(2026, 8, 24, 12, 15, 0, 0, time.UTC),
		},
		{
			PublicID:    "selected",
			State:       "success",
			CompletedAt: time.Date(2026, 8, 24, 12, 10, 0, 0, time.UTC),
		},
		{
			PublicID:    "older",
			State:       "success",
			CompletedAt: time.Date(2026, 8, 24, 11, 0, 0, 0, time.UTC),
		},
		{
			PublicID:    "pending",
			State:       "pending",
			CompletedAt: time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
		},
	}

	backup := BackupForRestorePoint(backups, restorePoint)
	c.Assert(backup, qt.IsNotNil)
	c.Assert(backup.PublicID, qt.Equals, "selected")
}

func TestBackupForRestorePoint_NoMatch(t *testing.T) {
	c := qt.New(t)

	restorePoint := time.Date(2026, 8, 24, 12, 14, 49, 0, time.UTC)
	backups := []*ps.Backup{
		{
			PublicID:    "too-new",
			State:       "success",
			CompletedAt: time.Date(2026, 8, 24, 12, 15, 0, 0, time.UTC),
		},
	}

	backup := BackupForRestorePoint(backups, restorePoint)
	c.Assert(backup, qt.IsNil)
}
