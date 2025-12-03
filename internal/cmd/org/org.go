package org

import (
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func OrgCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "org <command>",
		Short:             "List, show, and switch organizations",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.AddCommand(SwitchCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(ListCmd(ch))

	return cmd
}

// organization returns a table-serializable database model.
type organization struct {
	Name      string `header:"name" json:"name"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
	Current   bool   `header:"current" json:"current"`
}

// toOrgs returns a slice of printable orgs.
func toOrgs(organizations []*ps.Organization, currentOrgName string, format printer.Format) []*organization {
	orgs := make([]*organization, 0, len(organizations))

	for _, org := range organizations {
		orgs = append(orgs, toOrg(org, currentOrgName, format))
	}

	return orgs
}

func toOrg(org *ps.Organization, currentOrgName string, format printer.Format) *organization {
	isCurrent := org.Name == currentOrgName
	name := org.Name

	if isCurrent && format == printer.Human {
		name = "* " + org.Name
	}

	return &organization{
		Name:      name,
		CreatedAt: printer.GetMilliseconds(org.CreatedAt),
		UpdatedAt: printer.GetMilliseconds(org.UpdatedAt),
		Current:   isCurrent,
	}
}

func getCurrentOrganization(ch *cmdutil.Helper) string {
	if ch.Config.Organization != "" {
		return ch.Config.Organization
	}

	configPath, err := config.ProjectConfigPath()
	if err == nil {
		cfg, err := ch.ConfigFS.NewFileConfig(configPath)
		if err == nil && cfg.Organization != "" {
			return cfg.Organization
		}
	}

	configPath, err = config.DefaultConfigPath()
	if err == nil {
		cfg, err := ch.ConfigFS.NewFileConfig(configPath)
		if err == nil && cfg.Organization != "" {
			return cfg.Organization
		}
	}

	return ""
}
