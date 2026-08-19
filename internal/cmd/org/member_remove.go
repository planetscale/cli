package org

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func MemberRemoveCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		force               bool
		deletePasswords     bool
		deleteServiceTokens bool
	}

	cmd := &cobra.Command{
		Use:     "remove <email|user-id>",
		Short:   "Remove a member from an organization",
		Aliases: []string{"rm"},
		Long: `Remove a member from an organization.

Identify the member by email or by the USER_ID from 'org member list'.
Removing someone else requires organization admin. You can remove yourself
(leave) without being an admin. The last admin cannot be removed.`,
		Args: cmdutil.RequiredArgs("email|user-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			id := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching member %s in %s...", printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization)))
			member, err := resolveMember(ctx, ch, client, id)
			end()
			if err != nil {
				return err
			}

			if !flags.force {
				if err := ch.Printer.ConfirmCommand(member.User.Email, "remove organization member", "removal of organization member"); err != nil {
					return err
				}
			}

			end = ch.Printer.PrintProgress(fmt.Sprintf("Removing %s from %s...", printer.BoldBlue(member.User.Email), printer.BoldBlue(ch.Config.Organization)))
			defer end()

			err = client.Organizations.RemoveMember(ctx, &ps.RemoveOrganizationMemberRequest{
				Organization:        ch.Config.Organization,
				UserID:              member.User.ID,
				DeletePasswords:     flags.deletePasswords,
				DeleteServiceTokens: flags.deleteServiceTokens,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return memberNotFound(ch.Config.Organization, id)
				case ps.ErrPermission:
					// More than one server-side rule can reject this, so surface the
					// API's reason rather than assuming the caller is not an admin.
					return fmt.Errorf("cannot remove %s: %w", member.User.Email, err)
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Removed %s from %s.\n",
					printer.BoldBlue(member.User.Email),
					printer.BoldBlue(ch.Config.Organization))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result": "member removed",
				"org":    ch.Config.Organization,
				"user":   member.User.ID,
			})
		},
	}

	cmd.Flags().BoolVar(&flags.force, "force", false, "Remove the member without confirmation")
	cmd.Flags().BoolVar(&flags.deletePasswords, "delete-passwords", false, "Delete passwords created by the member. Cannot be used when removing yourself.")
	cmd.Flags().BoolVar(&flags.deleteServiceTokens, "delete-service-tokens", false, "Delete service tokens created by the member. Cannot be used when removing yourself.")
	return cmd
}
