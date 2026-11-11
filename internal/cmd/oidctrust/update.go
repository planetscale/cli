package oidctrust

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name                 string
		maxTTL               int
		conditions           []string
		orgAccesses          []string
		databaseAccesses     []string
		allDatabasesAccesses []string
	}

	cmd := &cobra.Command{
		Use:   "update <trust-id>",
		Short: "Update an OIDC trust",
		Args:  cmdutil.RequiredArgs("trust-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			f := cmd.Flags()

			req := &ps.UpdateOIDCTrustRequest{Organization: ch.Config.Organization, ID: id}
			changed := false

			if f.Changed("name") {
				req.DisplayName = &flags.name
				changed = true
			}
			if f.Changed("max-ttl") {
				req.MaxTTLSeconds = &flags.maxTTL
				changed = true
			}
			if f.Changed("condition") {
				conditions, err := parseConditions(nonEmpty(flags.conditions))
				if err != nil {
					return err
				}
				req.Conditions = &conditions
				changed = true
			}
			if f.Changed("org-access") || f.Changed("database-access") {
				accesses, err := parseAccesses(nonEmpty(flags.orgAccesses), nonEmpty(flags.databaseAccesses))
				if err != nil {
					return err
				}
				req.Accesses = &accesses
				changed = true
			}
			if f.Changed("all-databases-access") {
				bulk := nonEmpty(flags.allDatabasesAccesses)
				req.BulkDatabaseAccesses = &bulk
				changed = true
			}

			if !changed {
				return fmt.Errorf("at least one of --name, --max-ttl, --condition, --org-access, --database-access, or --all-databases-access must be provided")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating OIDC trust %s", printer.BoldBlue(id)))
			defer end()

			trust, err := client.OIDCTrusts.Update(cmd.Context(), req)
			if err != nil {
				return notFoundOrHandle(ch, id, err)
			}

			end()

			return ch.Printer.PrintResource(toOIDCTrust(trust))
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "The display name of the trust")
	cmd.Flags().IntVar(&flags.maxTTL, "max-ttl", 0, "The maximum token lifetime in seconds")
	cmd.Flags().StringArrayVar(&flags.conditions, "condition", nil, "A claim the token must match, as claim=value; repeatable; replaces all conditions")
	cmd.Flags().StringArrayVar(&flags.orgAccesses, "org-access", nil, "An organization access to grant; repeatable; replaces all org and database accesses")
	cmd.Flags().StringArrayVar(&flags.databaseAccesses, "database-access", nil, "A database access to grant, as database:access; repeatable; replaces all org and database accesses")
	cmd.Flags().StringArrayVar(&flags.allDatabasesAccesses, "all-databases-access", nil, "An access to grant on all databases; repeatable; replaces all all-databases accesses")

	return cmd
}
