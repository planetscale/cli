package org

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func MemberShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <email|user-id>",
		Short: "Show an organization member",
		Long: `Show an organization member by email or user id.

'org member list' prints both EMAIL and USER_ID.`,
		Args: cmdutil.RequiredArgs("email|user-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			id := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching member %s in %s...", printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization)))
			defer end()

			member, err := client.Organizations.GetMember(ctx, &ps.GetOrganizationMemberRequest{
				Organization: ch.Config.Organization,
				UserID:       id,
			})
			if err != nil {
				if cmdutil.ErrCode(err) != ps.ErrNotFound {
					return cmdutil.HandleError(err)
				}
				member, err = resolveMember(ctx, ch, client, id)
				if err != nil {
					return err
				}
			}
			end()

			return ch.Printer.PrintResource(toOrganizationMember(member))
		},
	}

	return cmd
}
