package org

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func MemberListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		query   string
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List members of an organization",
		Long: `List members of an organization.

Results are paginated: 100 members per page by default. Use --page and
--per-page to walk organizations with more members than one page holds.`,
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
				Query:        flags.query,
			}, ps.WithPage(flags.page), ps.WithPerPage(flags.perPage))
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
				if flags.page > 0 {
					ch.Printer.Println("No members found on this page.")
				} else if flags.query != "" {
					ch.Printer.Printf("No members in %s match %s.\n", printer.BoldBlue(org), printer.BoldBlue(flags.query))
				} else {
					ch.Printer.Printf("No members in %s.\n", printer.BoldBlue(org))
				}
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationMembers(members))
		},
	}

	cmd.Flags().StringVar(&flags.query, "query", "", "Filter members by name or email prefix")
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")
	return cmd
}
