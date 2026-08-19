package org

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func MemberUpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var role string

	cmd := &cobra.Command{
		Use:   "update <email|user-id>",
		Short: "Update an organization member's role",
		Long: `Update another member's organization role.

Identify the member by email or by the USER_ID from 'org member list'.
Only organization admins can do this. You cannot change your own role.
Assignable roles: admin, member, analyst.`,
		Args: cmdutil.RequiredArgs("email|user-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			id := args[0]

			if role == "" {
				return fmt.Errorf("must specify --role (admin, member, or analyst)")
			}
			switch role {
			case "admin", "member", "analyst":
			default:
				return fmt.Errorf("--role accepts admin, member, or analyst, got %q", role)
			}

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

			end = ch.Printer.PrintProgress(fmt.Sprintf("Updating role for %s to %s...", printer.BoldBlue(member.User.Email), printer.BoldBlue(role)))
			defer end()

			updated, err := client.Organizations.UpdateMember(ctx, &ps.UpdateOrganizationMemberRequest{
				Organization: ch.Config.Organization,
				UserID:       member.User.ID,
				Role:         role,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return memberNotFound(ch.Config.Organization, id)
				case ps.ErrPermission:
					// More than one server-side rule can reject this, so surface the
					// API's reason rather than assuming the caller is not an admin.
					return fmt.Errorf("cannot change the role for %s: %w", member.User.Email, err)
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Updated %s to %s in %s.\n",
					printer.BoldBlue(updated.User.Email),
					printer.BoldBlue(updated.Role),
					printer.BoldBlue(ch.Config.Organization))
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationMember(updated))
		},
	}

	cmd.Flags().StringVar(&role, "role", "", "Role to assign: admin, member, or analyst")
	return cmd
}
