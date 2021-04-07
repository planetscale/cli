package org

import (
	"context"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// organization returns a table-serializable database model.
type organization struct {
	Name      string `header:"name" json:"name"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`
}

func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List the currently active organizations",
		Args:    cobra.NoArgs,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			orgs, err := client.Organizations.List(ctx)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			if len(orgs) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No organizations exist\n")
				return nil
			}

			return ch.Printer.PrintResource(toOrgs(orgs))
		},
	}

	return cmd
}
