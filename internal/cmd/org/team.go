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

func TeamCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team <command>",
		Short: "Manage organization teams",
	}
	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org")
	cmd.AddCommand(TeamListCmd(ch))
	cmd.AddCommand(TeamShowCmd(ch))
	cmd.AddCommand(TeamCreateCmd(ch))
	cmd.AddCommand(TeamUpdateCmd(ch))
	cmd.AddCommand(TeamDeleteCmd(ch))
	cmd.AddCommand(TeamMemberCmd(ch))
	return cmd
}

type organizationTeam struct {
	ID          string `header:"id" json:"id"`
	Name        string `header:"name" json:"name"`
	Slug        string `header:"slug" json:"slug"`
	Description string `header:"description" json:"description"`
	Members     int    `header:"members" json:"members"`
	Managed     bool   `header:"managed" json:"managed"`
	orig        *ps.OrganizationTeam
}

func toOrganizationTeam(team *ps.OrganizationTeam) *organizationTeam {
	description := ""
	if team.Description != nil {
		description = *team.Description
	}
	return &organizationTeam{
		ID:          team.ID,
		Name:        team.Name,
		Slug:        team.Slug,
		Description: description,
		Members:     len(team.Members),
		Managed:     team.Managed,
		orig:        team,
	}
}

func toOrganizationTeams(teams []*ps.OrganizationTeam) []*organizationTeam {
	out := make([]*organizationTeam, 0, len(teams))
	for _, team := range teams {
		out = append(out, toOrganizationTeam(team))
	}
	return out
}

func (t *organizationTeam) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(t.orig, "", "  ")
}

func (t *organizationTeam) MarshalCSVValue() interface{} {
	return []*organizationTeam{t}
}

type organizationTeamMember struct {
	ID     string `header:"id" json:"id"`
	UserID string `header:"user_id" json:"user_id"`
	Name   string `header:"name" json:"name"`
	Email  string `header:"email" json:"email"`
	orig   *ps.OrganizationTeamMembership
}

func toOrganizationTeamMember(member *ps.OrganizationTeamMembership) *organizationTeamMember {
	name := member.User.DisplayName
	if name == "" {
		name = member.User.Name
	}
	return &organizationTeamMember{
		ID:     member.ID,
		UserID: member.User.ID,
		Name:   name,
		Email:  member.User.Email,
		orig:   member,
	}
}

func toOrganizationTeamMembers(members []*ps.OrganizationTeamMembership) []*organizationTeamMember {
	out := make([]*organizationTeamMember, 0, len(members))
	for _, member := range members {
		out = append(out, toOrganizationTeamMember(member))
	}
	return out
}

func (m *organizationTeamMember) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(m.orig, "", "  ")
}

func (m *organizationTeamMember) MarshalCSVValue() interface{} {
	return []*organizationTeamMember{m}
}

func teamNotFound(org, id string) error {
	return fmt.Errorf("team %s does not exist in organization %s", printer.BoldBlue(id), printer.BoldBlue(org))
}

func resolveTeam(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, id string) (*ps.OrganizationTeam, error) {
	org := ch.Config.Organization
	team, err := client.Organizations.GetTeam(ctx, &ps.GetOrganizationTeamRequest{
		Organization: org,
		Team:         id,
	})
	if err == nil {
		return team, nil
	}
	if cmdutil.ErrCode(err) != ps.ErrNotFound {
		return nil, cmdutil.HandleError(err)
	}

	const perPage = 100
	for _, query := range []string{id, ""} {
		page := 1
		for {
			teams, listErr := client.Organizations.ListTeams(ctx, &ps.ListOrganizationTeamsRequest{
				Organization: org,
				Query:        query,
			}, ps.WithPage(page), ps.WithPerPage(perPage))
			if listErr != nil {
				return nil, cmdutil.HandleError(listErr)
			}
			for _, candidate := range teams {
				if candidate.ID == id || strings.EqualFold(candidate.Name, id) || strings.EqualFold(candidate.Slug, id) {
					return candidate, nil
				}
			}
			if len(teams) < perPage {
				break
			}
			page++
		}
	}
	return nil, teamNotFound(org, id)
}

func TeamShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "show <team-id-or-name>",
		Short: "Show an organization team",
		Args:  cmdutil.RequiredArgs("team-id-or-name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}
			team, err := resolveTeam(cmd.Context(), ch, client, args[0])
			if err != nil {
				return err
			}
			return ch.Printer.PrintResource(toOrganizationTeam(team))
		},
	}
}

func TeamCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name        string
		description string
	}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an organization team",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.name == "" {
				return fmt.Errorf("must specify --name")
			}
			client, err := ch.Client()
			if err != nil {
				return err
			}
			team, err := client.Organizations.CreateTeam(cmd.Context(), &ps.CreateOrganizationTeamRequest{
				Organization: ch.Config.Organization,
				Name:         flags.name,
				Description:  flags.description,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Created team %s in %s.\n", printer.BoldBlue(team.Name), printer.BoldBlue(ch.Config.Organization))
				return nil
			}
			return ch.Printer.PrintResource(toOrganizationTeam(team))
		},
	}
	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the team")
	cmd.Flags().StringVar(&flags.description, "description", "", "Description of the team")
	cmd.MarkFlagRequired("name")
	return cmd
}

func TeamUpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name        string
		description string
	}
	cmd := &cobra.Command{
		Use:   "update <team>",
		Short: "Update an organization team",
		Args:  cmdutil.RequiredArgs("team"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("description") {
				return fmt.Errorf("must specify --name or --description")
			}
			client, err := ch.Client()
			if err != nil {
				return err
			}
			team, err := resolveTeam(cmd.Context(), ch, client, args[0])
			if err != nil {
				return err
			}
			req := &ps.UpdateOrganizationTeamRequest{
				Organization: ch.Config.Organization,
				Team:         team.Slug,
			}
			if cmd.Flags().Changed("name") {
				req.Name = &flags.name
			}
			if cmd.Flags().Changed("description") {
				req.Description = &flags.description
			}
			updated, err := client.Organizations.UpdateTeam(cmd.Context(), req)
			if err != nil {
				return cmdutil.HandleError(err)
			}
			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Updated team %s in %s.\n", printer.BoldBlue(updated.Name), printer.BoldBlue(ch.Config.Organization))
				return nil
			}
			return ch.Printer.PrintResource(toOrganizationTeam(updated))
		},
	}
	cmd.Flags().StringVar(&flags.name, "name", "", "New name for the team")
	cmd.Flags().StringVar(&flags.description, "description", "", "New description for the team")
	return cmd
}

func TeamDeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "delete <team>",
		Short:   "Delete an organization team",
		Aliases: []string{"rm"},
		Args:    cmdutil.RequiredArgs("team"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}
			team, err := resolveTeam(cmd.Context(), ch, client, args[0])
			if err != nil {
				return err
			}
			if !force {
				if err := ch.Printer.ConfirmCommand(team.Name, "delete organization team", "deletion of organization team"); err != nil {
					return err
				}
			}
			err = client.Organizations.DeleteTeam(cmd.Context(), &ps.DeleteOrganizationTeamRequest{
				Organization: ch.Config.Organization,
				Team:         team.Slug,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Deleted team %s from %s.\n", printer.BoldBlue(team.Name), printer.BoldBlue(ch.Config.Organization))
				return nil
			}
			return ch.Printer.PrintResource(map[string]string{
				"result": "team deleted",
				"org":    ch.Config.Organization,
				"team":   team.ID,
			})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Delete the team without confirmation")
	return cmd
}

func TeamMemberCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member <command>",
		Short: "Manage organization team members",
	}
	cmd.AddCommand(TeamMemberListCmd(ch))
	cmd.AddCommand(TeamMemberAddCmd(ch))
	cmd.AddCommand(TeamMemberRemoveCmd(ch))
	return cmd
}

func TeamMemberListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}
	cmd := &cobra.Command{
		Use:   "list <team>",
		Short: "List members of an organization team",
		Long: `List members of an organization team.

Results are paginated: 100 members per page by default. Use --page and
--per-page to walk teams with more members than one page holds.`,
		Aliases: []string{"ls"},
		Args:    cmdutil.RequiredArgs("team"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}
			team, err := resolveTeam(ctx, ch, client, args[0])
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching members of team %s...", printer.BoldBlue(team.Name)))
			defer end()

			members, err := client.Organizations.ListTeamMembers(ctx, &ps.ListOrganizationTeamMembersRequest{
				Organization: org,
				Team:         team.Slug,
			}, ps.WithPage(flags.page), ps.WithPerPage(flags.perPage))
			if err != nil {
				return cmdutil.HandleError(err)
			}
			end()

			if len(members) == 0 && ch.Printer.Format() == printer.Human {
				if flags.page > 0 {
					ch.Printer.Println("No team members found on this page.")
				} else {
					ch.Printer.Printf("No members in team %s.\n", printer.BoldBlue(team.Name))
				}
				return nil
			}
			return ch.Printer.PrintResource(toOrganizationTeamMembers(members))
		},
	}
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")
	return cmd
}

func TeamMemberAddCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "add <team> <email-or-id>",
		Short: "Add an organization member to a team",
		Args:  cmdutil.RequiredArgs("team", "email-or-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}
			team, err := resolveTeam(cmd.Context(), ch, client, args[0])
			if err != nil {
				return err
			}
			member, err := resolveMember(cmd.Context(), ch, client, args[1])
			if err != nil {
				return err
			}
			added, err := client.Organizations.AddTeamMember(cmd.Context(), &ps.AddOrganizationTeamMemberRequest{
				Organization: ch.Config.Organization,
				Team:         team.Slug,
				UserID:       member.User.ID,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Added %s to team %s.\n", printer.BoldBlue(member.User.Email), printer.BoldBlue(team.Name))
				return nil
			}
			return ch.Printer.PrintResource(toOrganizationTeamMember(added))
		},
	}
}

func resolveTeamMember(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, team, id string) (*ps.OrganizationTeamMembership, error) {
	page := 1
	const perPage = 100
	for {
		members, err := client.Organizations.ListTeamMembers(ctx, &ps.ListOrganizationTeamMembersRequest{
			Organization: ch.Config.Organization,
			Team:         team,
		}, ps.WithPage(page), ps.WithPerPage(perPage))
		if err != nil {
			return nil, cmdutil.HandleError(err)
		}
		for _, member := range members {
			if member.ID == id || member.User.ID == id || strings.EqualFold(member.User.Email, id) {
				return member, nil
			}
		}
		if len(members) < perPage {
			break
		}
		page++
	}
	return nil, fmt.Errorf("member %s does not belong to team %s", printer.BoldBlue(id), printer.BoldBlue(team))
}

func TeamMemberRemoveCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		force           bool
		deletePasswords bool
	}
	cmd := &cobra.Command{
		Use:     "remove <team> <email-or-id>",
		Short:   "Remove a member from an organization team",
		Aliases: []string{"rm"},
		Args:    cmdutil.RequiredArgs("team", "email-or-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}
			team, err := resolveTeam(cmd.Context(), ch, client, args[0])
			if err != nil {
				return err
			}
			member, err := resolveTeamMember(cmd.Context(), ch, client, team.Slug, args[1])
			if err != nil {
				return err
			}
			if !flags.force {
				if err := ch.Printer.ConfirmCommand(member.User.Email, "remove organization team member", "removal of organization team member"); err != nil {
					return err
				}
			}
			err = client.Organizations.RemoveTeamMember(cmd.Context(), &ps.RemoveOrganizationTeamMemberRequest{
				Organization:    ch.Config.Organization,
				Team:            team.Slug,
				ID:              member.ID,
				DeletePasswords: flags.deletePasswords,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Removed %s from team %s.\n", printer.BoldBlue(member.User.Email), printer.BoldBlue(team.Name))
				return nil
			}
			return ch.Printer.PrintResource(map[string]string{
				"result": "team member removed",
				"org":    ch.Config.Organization,
				"team":   team.ID,
				"user":   member.User.ID,
			})
		},
	}
	cmd.Flags().BoolVar(&flags.force, "force", false, "Remove the team member without confirmation")
	cmd.Flags().BoolVar(&flags.deletePasswords, "delete-passwords", false, "Delete passwords created through this team")
	return cmd
}
