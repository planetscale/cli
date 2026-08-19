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

func MemberCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member <command>",
		Short: "List, show, update, and remove organization members",
		Long: `Manage organization members and their roles.

show, update, and remove take an email or a user id (the USER_ID column from
'org member list'). Email is usually the easiest.

Only organization admins can change another member's role or remove someone
else. Nobody can change their own role. Members can still leave the organization
themselves.`,
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org")

	cmd.AddCommand(MemberListCmd(ch))
	cmd.AddCommand(MemberShowCmd(ch))
	cmd.AddCommand(MemberUpdateCmd(ch))
	cmd.AddCommand(MemberRemoveCmd(ch))

	return cmd
}

type organizationMember struct {
	UserID string `header:"user_id" json:"user_id"`
	Name   string `header:"name" json:"name"`
	Email  string `header:"email" json:"email"`
	Role   string `header:"role" json:"role"`

	orig *ps.OrganizationMembership
}

func toOrganizationMembers(members []*ps.OrganizationMembership) []*organizationMember {
	out := make([]*organizationMember, 0, len(members))
	for _, m := range members {
		out = append(out, toOrganizationMember(m))
	}
	return out
}

func toOrganizationMember(m *ps.OrganizationMembership) *organizationMember {
	name := m.User.DisplayName
	if name == "" {
		name = m.User.Name
	}
	return &organizationMember{
		UserID: m.User.ID,
		Name:   name,
		Email:  m.User.Email,
		Role:   m.Role,
		orig:   m,
	}
}

func (m *organizationMember) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(m.orig, "", "  ")
}

func (m *organizationMember) MarshalCSVValue() interface{} {
	return []*organizationMember{m}
}

func memberNotFound(org, id string) error {
	return fmt.Errorf("member %s does not exist in organization %s",
		printer.BoldBlue(id), printer.BoldBlue(org))
}

func matchMember(m *ps.OrganizationMembership, id string) bool {
	if m.User.ID == id || m.ID == id {
		return true
	}
	return strings.EqualFold(m.User.Email, id)
}

func resolveMember(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, id string) (*ps.OrganizationMembership, error) {
	org := ch.Config.Organization
	query := ""
	if strings.Contains(id, "@") {
		query = id
	}

	page := 1
	perPage := 100
	for {
		members, err := client.Organizations.ListMembers(ctx, &ps.ListOrganizationMembersRequest{
			Organization: org,
			Query:        query,
		}, ps.WithPage(page), ps.WithPerPage(perPage))
		if err != nil {
			switch cmdutil.ErrCode(err) {
			case ps.ErrNotFound:
				return nil, fmt.Errorf("organization %s does not exist", printer.BoldBlue(org))
			default:
				return nil, cmdutil.HandleError(err)
			}
		}

		for _, m := range members {
			if matchMember(m, id) {
				return m, nil
			}
		}
		if len(members) < perPage {
			break
		}
		page++
	}

	return nil, memberNotFound(org, id)
}
