package org

import (
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
		Short:   "Manage organization custom roles",
		Aliases: []string{"custom-roles"},
	}
	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org")
	cmd.AddCommand(CustomRoleListCmd(ch))
	cmd.AddCommand(CustomRoleShowCmd(ch))
	cmd.AddCommand(CustomRoleCreateCmd(ch))
	cmd.AddCommand(CustomRoleUpdateCmd(ch))
	cmd.AddCommand(CustomRoleDeleteCmd(ch))
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

func customRoleNotFound(org, slug string) error {
	return fmt.Errorf("custom role %s does not exist in organization %s", printer.BoldBlue(slug), printer.BoldBlue(org))
}

func handleCustomRoleError(org, slug string, err error) error {
	switch cmdutil.ErrCode(err) {
	case ps.ErrNotFound:
		return customRoleNotFound(org, slug)
	default:
		return cmdutil.HandleError(err)
	}
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
		Long: `Show an organization custom role by slug.

'org custom-role list' prints SLUG.`,
		Args: cmdutil.RequiredArgs("slug"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			org := ch.Config.Organization
			slug := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			const perPage = 100
			page := 1
			for {
				roles, err := client.Organizations.ListCustomRoles(ctx, &ps.ListOrganizationCustomRolesRequest{
					Organization: org,
				}, ps.WithPage(page), ps.WithPerPage(perPage))
				if err != nil {
					switch cmdutil.ErrCode(err) {
					case ps.ErrNotFound:
						return fmt.Errorf("organization %s does not exist", printer.BoldBlue(org))
					default:
						return cmdutil.HandleError(err)
					}
				}
				for _, role := range roles {
					if role.Slug == slug {
						return ch.Printer.PrintResource(toOrganizationCustomRole(role))
					}
				}
				if len(roles) < perPage {
					break
				}
				page++
			}
			return customRoleNotFound(org, slug)
		},
	}
}

func CustomRoleCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name        string
		description string
		permissions []string
	}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an organization custom role",
		Long: `Create an organization custom role.

The slug is assigned from the name. --permissions replaces the full set and
must include show_organization.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.name == "" {
				return fmt.Errorf("must specify --name")
			}
			if len(flags.permissions) == 0 {
				return fmt.Errorf("must specify --permissions")
			}
			client, err := ch.Client()
			if err != nil {
				return err
			}
			role, err := client.Organizations.CreateCustomRole(cmd.Context(), &ps.CreateOrganizationCustomRoleRequest{
				Organization: ch.Config.Organization,
				Name:         flags.name,
				Description:  flags.description,
				Permissions:  flags.permissions,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Created custom role %s in %s.\n", printer.BoldBlue(role.Slug), printer.BoldBlue(ch.Config.Organization))
				return nil
			}
			return ch.Printer.PrintResource(toOrganizationCustomRole(role))
		},
	}
	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the custom role")
	cmd.Flags().StringVar(&flags.description, "description", "", "Description of the custom role")
	cmd.Flags().StringSliceVar(&flags.permissions, "permissions", nil, "Actions the role grants; must include show_organization. Repeat or comma-separate")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("permissions")
	return cmd
}

func CustomRoleUpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name        string
		description string
		permissions []string
	}
	cmd := &cobra.Command{
		Use:   "update <slug>",
		Short: "Update an organization custom role",
		Long: `Update an organization custom role by slug.

--permissions replaces the full set when provided and must include
show_organization.`,
		Args: cmdutil.RequiredArgs("slug"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("description") && !cmd.Flags().Changed("permissions") {
				return fmt.Errorf("must specify --name, --description, or --permissions")
			}
			client, err := ch.Client()
			if err != nil {
				return err
			}
			req := &ps.UpdateOrganizationCustomRoleRequest{
				Organization: ch.Config.Organization,
				Slug:         args[0],
			}
			if cmd.Flags().Changed("name") {
				req.Name = &flags.name
			}
			if cmd.Flags().Changed("description") {
				req.Description = &flags.description
			}
			if cmd.Flags().Changed("permissions") {
				req.Permissions = &flags.permissions
			}
			updated, err := client.Organizations.UpdateCustomRole(cmd.Context(), req)
			if err != nil {
				return handleCustomRoleError(ch.Config.Organization, args[0], err)
			}
			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Updated custom role %s in %s.\n", printer.BoldBlue(updated.Slug), printer.BoldBlue(ch.Config.Organization))
				return nil
			}
			return ch.Printer.PrintResource(toOrganizationCustomRole(updated))
		},
	}
	cmd.Flags().StringVar(&flags.name, "name", "", "New name for the custom role")
	cmd.Flags().StringVar(&flags.description, "description", "", "New description for the custom role")
	cmd.Flags().StringSliceVar(&flags.permissions, "permissions", nil, "Actions the role grants; must include show_organization. Repeat or comma-separate. Replaces the full set")
	return cmd
}

func CustomRoleDeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "delete <slug>",
		Short:   "Delete an organization custom role",
		Aliases: []string{"rm"},
		Args:    cmdutil.RequiredArgs("slug"),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !force {
				if err := ch.Printer.ConfirmCommand(slug, "delete organization custom role", "deletion of organization custom role"); err != nil {
					return err
				}
			}
			client, err := ch.Client()
			if err != nil {
				return err
			}
			err = client.Organizations.DeleteCustomRole(cmd.Context(), &ps.DeleteOrganizationCustomRoleRequest{
				Organization: ch.Config.Organization,
				Slug:         slug,
			})
			if err != nil {
				return handleCustomRoleError(ch.Config.Organization, slug, err)
			}
			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Deleted custom role %s from %s.\n", printer.BoldBlue(slug), printer.BoldBlue(ch.Config.Organization))
				return nil
			}
			return ch.Printer.PrintResource(map[string]string{
				"result": "custom role deleted",
				"org":    ch.Config.Organization,
				"slug":   slug,
			})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Delete the custom role without confirmation")
	return cmd
}
