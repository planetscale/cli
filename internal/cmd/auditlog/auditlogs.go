package auditlog

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lensesio/tableprinter"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

// AuditLogCmd handles audit logs.
func AuditLogCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "audit-log <command>",
		Short:             "List audit logs",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(ch))

	return cmd
}

type AuditLogs []*AuditLog

type AuditLog struct {
	Actor     string `header:"actor" json:"actor"`
	Action    string `header:"action" json:"action"`
	Event     string `header:"event" json:"type"`
	RemoteIP  string `header:"remote_ip" json:"remote_ip"`
	Location  string `header:"location" json:"location"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`

	orig *ps.AuditLog
}

func (a *AuditLog) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(a.orig, "", "  ")
}

func (a *AuditLog) MarshalCSVValue() interface{} {
	return []*AuditLog{a}
}

func (a AuditLogs) String() string {
	var buf strings.Builder
	tableprinter.Print(&buf, a)
	return buf.String()
}

// toAuditLog Returns a struct that prints out the various fields of a branch model.
func toAuditLog(a *ps.AuditLog) *AuditLog {
	return &AuditLog{
		Actor:     a.ActorDisplayName,
		Action:    fmt.Sprintf("%s %s", strings.Title(a.Action), a.AuditableDisplayName),
		Event:     a.AuditAction,
		RemoteIP:  a.RemoteIP,
		Location:  a.Location,
		CreatedAt: a.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		orig:      a,
	}
}

func toAuditLogs(audilogs []*ps.AuditLog) []*AuditLog {
	al := make([]*AuditLog, 0, len(audilogs))
	for _, audilog := range audilogs {
		al = append(al, toAuditLog(audilog))
	}
	return al
}
