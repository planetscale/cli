package cmdutil

import (
	"context"
	"fmt"
	"time"

	ps "github.com/planetscale/cli/internal/planetscale"
)

func BackupForRestorePoint(backups []*ps.Backup, restorePoint time.Time) *ps.Backup {
	target := restorePoint.UnixMilli()
	var selected *ps.Backup

	for _, backup := range backups {
		if backup.State != "success" || backup.CompletedAt.IsZero() {
			continue
		}

		completed := backup.CompletedAt.UnixMilli()
		if completed > target {
			continue
		}

		if selected != nil && !selected.CompletedAt.IsZero() && selected.CompletedAt.UnixMilli() >= completed {
			continue
		}

		selected = backup
	}

	return selected
}

func BackupIDForRestorePoint(ctx context.Context, client *ps.Client, org, database, branch, restorePoint string) (string, error) {
	restoreTime, err := time.Parse(time.RFC3339, restorePoint)
	if err != nil {
		return "", fmt.Errorf("invalid --restore-point timestamp %q: use RFC3339 format (e.g. 2023-01-01T00:00:00Z)", restorePoint)
	}

	restoreUTC := restoreTime.UTC()
	to := time.Date(restoreUTC.Year(), restoreUTC.Month(), restoreUTC.Day(), restoreUTC.Hour(), restoreUTC.Minute(), 59, 999000000, time.UTC)
	backups, err := client.Backups.List(ctx, &ps.ListBackupsRequest{
		Organization: org,
		Database:     database,
		Branch:       branch,
		To:           to.Format("2006-01-02T15:04:05.999Z"),
		State:        "success",
	})
	if err != nil {
		return "", HandleError(err)
	}

	backup := BackupForRestorePoint(backups, restoreTime)
	if backup == nil {
		return "", fmt.Errorf("no successful backup found for restore point %s on branch %s", restorePoint, branch)
	}

	return backup.PublicID, nil
}
