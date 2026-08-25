package branch

import (
	"fmt"
	"time"

	ps "github.com/planetscale/cli/internal/planetscale"
)

func backupForRestorePoint(backups []*ps.Backup, restorePoint time.Time) (*ps.Backup, error) {
	var selected *ps.Backup

	for _, backup := range backups {
		if backup.State != "success" || backup.CompletedAt.IsZero() {
			continue
		}

		if !backup.CompletedAt.Before(restorePoint) {
			continue
		}

		if selected == nil || backup.CompletedAt.After(selected.CompletedAt) {
			selected = backup
		}
	}

	if selected == nil {
		return nil, fmt.Errorf("no successful backup found for restore point %s", restorePoint.Format(time.RFC3339))
	}

	return selected, nil
}
