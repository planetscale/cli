package org

import (
	"context"
	"time"

	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// organization returns a table-serializable database model.
type organization struct {
	Name      string `header:"name" json:"name"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
}

func ListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List the currently active organizations",
		Args:    cobra.NoArgs,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			orgs, err := client.Organizations.List(ctx)
			if err != nil {
				return err
			}

			err = printer.PrintOutput(cfg.OutputJSON, &printer.ObjectPrinter{
				Source:  orgs,
				Printer: newOrganizationSlicePrinter(orgs),
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// newOrganizationSlicePrinter returns a slice of printable orgs.
func newOrganizationSlicePrinter(organizations []*ps.Organization) []*organization {
	orgs := make([]*organization, 0, len(organizations))

	for _, org := range organizations {
		orgs = append(orgs, newOrgPrinter(org))
	}

	return orgs
}

func newOrgPrinter(org *ps.Organization) *organization {
	return &organization{
		Name:      org.Name,
		CreatedAt: org.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		UpdatedAt: org.UpdatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
	}
}
