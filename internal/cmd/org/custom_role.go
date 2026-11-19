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

func CustomRoleCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "custom-role <command>",
		Short:   "List and show organization custom roles",
		Aliases: []string{"custom-roles"},
	}
	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org")
	cmd.AddCommand(CustomRoleListCmd(ch))
	cmd.AddCommand(CustomRoleShowCmd(ch))
	return cmd
}

type organizationCustomRole struct {
	Slug        string `header:"slug" json:"slug"`
	Name        string `header:"name" json:"name"`
	Description string `header:"description" json:"description"`
	Permissions string `header:"permissions" json:"permissions"`
	orig        *ps.OrganizationCustomRole
}

func toOrganizationCustomRole(role *ps.OrganizationCustomRole) *organizationCustomRole {
	description := ""
	if role.Description != nil {
		description = *role.Description
	}
	actions := make([]string, 0, len(role.Permissions))
	for _, permission := range role.Permissions {
		actions = append(actions, permission.Action)
	}
	return &organizationCustomRole{
		Slug:        role.Slug,
		Name:        role.Name,
		Description: description,
		Permissions: strings.Join(actions, ", "),
		orig:        role,
	}
}

func toOrganizationCustomRoles(roles []*ps.OrganizationCustomRole) []*organizationCustomRole {
	out := make([]*organizationCustomRole, 0, len(roles))
	for _, role := range roles {
		out = append(out, toOrganizationCustomRole(role))
	}
	return out
}

func (r *organizationCustomRole) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *organizationCustomRole) MarshalCSVValue() interface{} {
	return []*organizationCustomRole{r}
}

func customRoleNotFound(org, id string) error {
	return fmt.Errorf("custom role %s does not exist in organization %s", printer.BoldBlue(id), printer.BoldBlue(org))
}

func matchCustomRole(role *ps.OrganizationCustomRole, id string) bool {
	return role.Slug == id || role.ID == id || strings.EqualFold(role.Name, id)
}

func resolveCustomRole(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, id string) (*ps.OrganizationCustomRole, error) {
	org := ch.Config.Organization
	const perPage = 100
	page := 1
	for {
		roles, err := client.Organizations.ListCustomRoles(ctx, &ps.ListOrganizationCustomRolesRequest{
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
		for _, role := range roles {
			if matchCustomRole(role, id) {
				return role, nil
			}
		}
		if len(roles) < perPage {
			break
		}
		page++
	}
	return nil, customRoleNotFound(org, id)
}

func CustomRoleListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List custom roles in an organization",
		Long: `List custom roles in an organization.

Results are paginated: 100 roles per page by default. Use --page and
--per-page to walk organizations with more roles than one page holds.`,
		Args:    cobra.NoArgs,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching custom roles of %s...", printer.BoldBlue(org)))
			defer end()

			roles, err := client.Organizations.ListCustomRoles(ctx, &ps.ListOrganizationCustomRolesRequest{
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

			if len(roles) == 0 && ch.Printer.Format() == printer.Human {
				if flags.page > 0 {
					ch.Printer.Println("No custom roles found on this page.")
				} else {
					ch.Printer.Printf("No custom roles in %s.\n", printer.BoldBlue(org))
				}
				return nil
			}

			return ch.Printer.PrintResource(toOrganizationCustomRoles(roles))
		},
	}

	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")
	return cmd
}

func CustomRoleShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "show <slug>",
		Short: "Show an organization custom role",
		Long: `Show an organization custom role by slug, name, or id.

'org custom-role list' prints SLUG and NAME.`,
		Args: cmdutil.RequiredArgs("slug"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ch.Client()
			if err != nil {
				return err
			}
			role, err := resolveCustomRole(cmd.Context(), ch, client, args[0])
			if err != nil {
				return err
			}
			return ch.Printer.PrintResource(toOrganizationCustomRole(role))
		},
	}
}
