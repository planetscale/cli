package org

import (
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func OrgCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "org <command>",
		Short:             "Modify and manage organization options",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.AddCommand(SwitchCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(ListCmd(ch))

	return cmd
}

// toOrgs returns a slice of printable orgs.
func toOrgs(organizations []*ps.Organization) []*organization {
	orgs := make([]*organization, 0, len(organizations))

	for _, org := range organizations {
		orgs = append(orgs, toOrg(org))
	}

	return orgs
}

func toOrg(org *ps.Organization) *organization {
	return &organization{
		Name:      org.Name,
		CreatedAt: org.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt: org.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
	}
}
