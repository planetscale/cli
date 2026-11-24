package org

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func InviteCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "invite <command>",
		Short:   "List, send, and cancel organization invitations",
		Aliases: []string{"invitations"},
		Long: `Manage pending organization invitations.

--role is a built-in name (admin, member, analyst) or a custom role slug.
Service tokens can only invite people as members.`,
	}
	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org")
	cmd.AddCommand(InviteListCmd(ch))
	cmd.AddCommand(InviteCreateCmd(ch))
	cmd.AddCommand(InviteCancelCmd(ch))
	return cmd
}

type organizationInvitation struct {
	ID      string `header:"id" json:"id"`
	Email   string `header:"email" json:"email"`
	Role    string `header:"role" json:"role"`
	Expired bool   `header:"expired" json:"expired"`
	orig    *ps.OrganizationInvitation
}

func toOrganizationInvitation(invitation *ps.OrganizationInvitation) *organizationInvitation {
	return &organizationInvitation{
		ID:      invitation.ID,
		Email:   invitation.Email,
		Role:    invitation.Role,
		Expired: invitation.Expired,
		orig:    invitation,
	}
}

func toOrganizationInvitations(invitations []*ps.OrganizationInvitation) []*organizationInvitation {
	out := make([]*organizationInvitation, 0, len(invitations))
	for _, invitation := range invitations {
		out = append(out, toOrganizationInvitation(invitation))
	}
	return out
}

func (i *organizationInvitation) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(i.orig, "", "  ")
}

func (i *organizationInvitation) MarshalCSVValue() interface{} {
	return []*organizationInvitation{i}
}

func invitationNotFound(org, id string) error {
	return fmt.Errorf("invitation %s does not exist in organization %s", printer.BoldBlue(id), printer.BoldBlue(org))
}

func matchInvitation(invitation *ps.OrganizationInvitation, id string) bool {
	return invitation.ID == id || strings.EqualFold(invitation.Email, id)
}

func resolveInvitation(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, id string) (*ps.OrganizationInvitation, error) {
	org := ch.Config.Organization
	const perPage = 100
	page := 1
	for {
		invitations, err := client.Organizations.ListInvitations(ctx, &ps.ListOrganizationInvitationsRequest{
			Organization: org,
		}, ps.WithPage(page), ps.WithPerPage(perPage))
		if err != nil {
			switch cmdutil.ErrCode(err) {
			case ps.ErrNotFound:
				return nil, fmt.Errorf("organization %s does not exist", printer.BoldBlue(org))
			default:
				return nil, cmdutil.HandleError(err)
			}
		}
		for _, invitation := range invitations {
			if matchInvitation(invitation, id) {
				return invitation, nil
			}
		}
		if len(invitations) < perPage {
			break
		}
		page++
	}
	return nil, invitationNotFound(org, id)
}

func InviteListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pending invitations for an organization",
		Long: `List pending invitations for an organization.

Results are paginated: 100 invitations per page by default. Use --page and
--per-page to walk organizations with more invitations than one page holds.`,
		Args:    cobra.NoArgs,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching invitations of %s...", printer.BoldBlue(org)))
			defer end()

			invitations, err := client.Organizations.ListInvitations(ctx, &ps.ListOrganizationInvitationsRequest{
				Organization: org,
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

			if len(invitations) == 0 && ch.Printer.Format() == printer.Human {
				if flags.page > 0 {
					ch.Printer.Println("No invitations found on this page.")
				} else {
					ch.Printer.Printf("No pending invitations in %s.\n", printer.BoldBlue(org))
				}
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationInvitations(invitations))
		},
	}

	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")
	return cmd
}

func InviteCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var role string

	cmd := &cobra.Command{
		Use:   "create <email>",
		Short: "Send an organization invitation",
		Long: `Send an invitation to join the organization.

--role is a built-in name (admin, member, analyst) or a custom role slug.
Defaults to member. Service tokens can only invite people as members.`,
		Args: cmdutil.RequiredArgs("email"),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Inviting %s to %s as %s...", printer.BoldBlue(email), printer.BoldBlue(ch.Config.Organization), printer.BoldBlue(role)))
			defer end()

			invitation, err := client.Organizations.CreateInvitation(cmd.Context(), &ps.CreateOrganizationInvitationRequest{
				Organization: ch.Config.Organization,
				Email:        email,
				Role:         role,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("organization %s does not exist", printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Invited %s to %s as %s.\n",
					printer.BoldBlue(invitation.Email),
					printer.BoldBlue(ch.Config.Organization),
					printer.BoldBlue(invitation.Role))
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationInvitation(invitation))
		},
	}

	cmd.Flags().StringVar(&role, "role", "member", "Role to grant: a built-in name (admin, member, analyst) or a custom role slug")
	return cmd
}

func InviteCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "cancel <invitation-id>",
		Short:   "Cancel a pending organization invitation",
		Aliases: []string{"delete", "rm"},
		Args:    cmdutil.RequiredArgs("invitation-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			invitation, err := resolveInvitation(cmd.Context(), ch, client, id)
			if err != nil {
				return err
			}

			if !force {
				if err := ch.Printer.ConfirmCommand(invitation.Email, "cancel organization invitation", "cancellation of organization invitation"); err != nil {
					return err
				}
			}

			err = client.Organizations.CancelInvitation(cmd.Context(), &ps.CancelOrganizationInvitationRequest{
				Organization: ch.Config.Organization,
				ID:           invitation.ID,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return invitationNotFound(ch.Config.Organization, id)
				default:
					return cmdutil.HandleError(err)
				}
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Canceled invitation for %s in %s.\n",
					printer.BoldBlue(invitation.Email),
					printer.BoldBlue(ch.Config.Organization))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result":     "invitation canceled",
				"org":        ch.Config.Organization,
				"invitation": invitation.ID,
				"email":      invitation.Email,
			})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Cancel the invitation without confirmation")
	return cmd
}
