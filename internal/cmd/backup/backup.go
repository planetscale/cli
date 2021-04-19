package backup

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lensesio/tableprinter"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

// BackupCmd handles branch backups.
func BackupCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "backup <command>",
		Short:             "Create, read, destroy, and update branch backups",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))
	cmd.AddCommand(ShowCmd(ch))

	return cmd
}

type Backups []*Backup

type Backup struct {
	Name        string `header:"name" json:"name"`
	State       string `header:"state" json:"state"`
	Size        int64  `header:"size" json:"size"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt   int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
	StartedAt   int64  `header:"started_at,timestamp(ms|utc|human)" json:"started_at"`
	ExpiresAt   int64  `header:"expires_at,timestamp(ms|utc|human)" json:"expires_at"`
	CompletedAt int64  `header:"completed_at,timestamp(ms|utc|human)" json:"completed_at"`

	orig *ps.Backup
}

func (b *Backup) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(b.orig, "", "  ")
}

func (b *Backup) MarshalCSVValue() interface{} {
	return []*Backup{b}
}

func (b Backups) String() string {
	var buf strings.Builder
	tableprinter.Print(&buf, b)
	return buf.String()
}

// toBackup Returns a struct that prints out the various fields of a branch model.
func toBackup(backup *ps.Backup) *Backup {
	return &Backup{
		Name:        backup.Name,
		State:       backup.State,
		Size:        backup.Size,
		CreatedAt:   backup.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt:   backup.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		StartedAt:   backup.StartedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		ExpiresAt:   backup.ExpiresAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		CompletedAt: backup.CompletedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		orig:        backup,
	}
}

func toBackups(backups []*ps.Backup) []*Backup {
	bs := make([]*Backup, 0, len(backups))
	for _, backup := range backups {
		bs = append(bs, toBackup(backup))
	}
	return bs
}
