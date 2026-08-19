package org

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func MemberListCmd(ch *cmdutil.Helper) *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List members of an organization",
		Args:    cobra.NoArgs,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching members of %s...", printer.BoldBlue(org)))
			defer end()

			members, err := client.Organizations.ListMembers(ctx, &ps.ListOrganizationMembersRequest{
				Organization: org,
				Query:        query,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("organization %s does not exist", printer.BoldBlue(org))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(members) == 0 && ch.Printer.Format() == printer.Human {
				if query != "" {
					ch.Printer.Printf("No members in %s match %s.\n", printer.BoldBlue(org), printer.BoldBlue(query))
				} else {
					ch.Printer.Printf("No members in %s.\n", printer.BoldBlue(org))
				}
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationMembers(members))
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "Filter members by name or email prefix")
	return cmd
}
