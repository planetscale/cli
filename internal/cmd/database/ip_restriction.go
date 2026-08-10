package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// IPRestrictionCmd manages Postgres IP restrictions for a database.
func IPRestrictionCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ip-restriction <command>",
		Aliases: []string{"cidr", "cidrs"},
		Short:   "Manage Postgres IP restrictions",
		Long: `Manage IP restrictions for a PostgreSQL database.

IP restrictions limit which IPv4 addresses or ranges can connect. Rules can
optionally be scoped to a specific Postgres schema and/or role. This command
is only available for PostgreSQL databases.`,
	}

	cmd.AddCommand(IPRestrictionListCmd(ch))
	cmd.AddCommand(IPRestrictionShowCmd(ch))
	cmd.AddCommand(IPRestrictionCreateCmd(ch))
	cmd.AddCommand(IPRestrictionUpdateCmd(ch))
	cmd.AddCommand(IPRestrictionDeleteCmd(ch))

	return cmd
}

// IPRestrictionEntry is the human/JSON/CSV view of a Postgres IP restriction entry.
type IPRestrictionEntry struct {
	ID          string `header:"id" json:"id"`
	Schema      string `header:"schema" json:"schema"`
	Role        string `header:"role" json:"role"`
	CIDRs       string `header:"cidrs" json:"cidrs"`
	Description string `header:"description" json:"description"`
	Actor       string `header:"actor" json:"actor"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt   int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.PostgresCIDR
}

func (e *IPRestrictionEntry) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(e.orig, "", "  ")
}

func (e *IPRestrictionEntry) MarshalCSVValue() interface{} {
	return []*IPRestrictionEntry{e}
}

func toIPRestrictionEntry(entry *ps.PostgresCIDR) *IPRestrictionEntry {
	out := &IPRestrictionEntry{
		ID:        entry.ID,
		Schema:    emptyAsDash(entry.Schema),
		Role:      emptyAsDash(entry.Role),
		CIDRs:     strings.Join(entry.CIDRs, ", "),
		Actor:     emptyAsDash(entry.Actor.Name),
		CreatedAt: printer.GetMilliseconds(entry.CreatedAt),
		UpdatedAt: printer.GetMilliseconds(entry.UpdatedAt),
		orig:      entry,
	}
	if entry.Description != nil && *entry.Description != "" {
		out.Description = *entry.Description
	} else {
		out.Description = "-"
	}
	return out
}

func toIPRestrictionEntries(entries []*ps.PostgresCIDR) []*IPRestrictionEntry {
	out := make([]*IPRestrictionEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, toIPRestrictionEntry(entry))
	}
	return out
}

// requirePostgresDatabase fetches the database and errors if it is not PostgreSQL.
func requirePostgresDatabase(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, database string) error {
	db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
		Organization: ch.Config.Organization,
		Database:     database,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case ps.ErrNotFound:
			return fmt.Errorf("database %s does not exist in organization %s",
				printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
		default:
			return cmdutil.HandleError(err)
		}
	}
	if db.Kind != ps.DatabaseEnginePostgres {
		return fmt.Errorf("IP restrictions are only available for PostgreSQL databases; %s is %s",
			printer.BoldBlue(database), printer.BoldBlue(string(db.Kind)))
	}
	return nil
}
