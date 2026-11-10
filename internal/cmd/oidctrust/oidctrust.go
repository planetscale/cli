package oidctrust

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// OIDCTrustCmd encapsulates the command for managing OIDC trusts.
func OIDCTrustCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "oidc-trust <command>",
		Short:             "Manage OIDC trusts",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ProvidersCmd(ch))
	cmd.AddCommand(ShowCmd(ch))

	return cmd
}

// OIDCTrust returns a table and json serializable OIDC trust for printing.
type OIDCTrust struct {
	ID         string `header:"id" json:"id"`
	Name       string `header:"name" json:"display_name"`
	Provider   string `header:"provider" json:"provider"`
	Audience   string `header:"audience" json:"audience"`
	MaxTTL     int    `header:"max ttl" json:"max_ttl_seconds"`
	Conditions string `header:"conditions" json:"conditions"`
	Accesses   string `header:"accesses" json:"accesses"`
	CreatedAt  int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`

	orig *ps.OIDCTrust
}

func (t *OIDCTrust) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(t.orig, "", "  ")
}

func toOIDCTrust(t *ps.OIDCTrust) *OIDCTrust {
	conditions := make([]string, 0, len(t.Conditions))
	for _, c := range t.Conditions {
		conditions = append(conditions, c.Claim+"="+c.Value)
	}

	accesses := make([]string, 0, len(t.Accesses)+len(t.BulkDatabaseAccesses))
	for _, a := range t.Accesses {
		if a.ResourceType == "Database" {
			accesses = append(accesses, fmt.Sprintf("%s:%s", a.ResourceName, a.Access))
		} else {
			accesses = append(accesses, a.Access)
		}
	}
	for _, a := range t.BulkDatabaseAccesses {
		accesses = append(accesses, "*:"+a)
	}

	return &OIDCTrust{
		ID:         t.ID,
		Name:       t.DisplayName,
		Provider:   t.Provider,
		Audience:   t.Audience,
		MaxTTL:     t.MaxTTLSeconds,
		Conditions: strings.Join(conditions, ","),
		Accesses:   strings.Join(accesses, ","),
		CreatedAt:  printer.GetMilliseconds(t.CreatedAt),
		orig:       t,
	}
}
