package database

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// IPRestrictionUpdateCmd updates an IP restriction entry.
func IPRestrictionUpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		cidrs       []string
		schema      string
		role        string
		description string
	}

	cmd := &cobra.Command{
		Use:   "update <database> <entry-id>",
		Short: "Update an IP restriction entry",
		Long: `Update an IP restriction entry for a PostgreSQL database.

Only provided flags are changed. Pass an empty --description to clear it.
Pass an empty --schema or --role to allow access for all schemas or roles.`,
		Args: cmdutil.RequiredArgs("database", "entry-id"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			entryID := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := requirePostgresDatabase(ctx, ch, client, database); err != nil {
				return err
			}

			req := &ps.UpdatePostgresCIDRRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				ID:           entryID,
			}

			changed := false
			if cmd.Flags().Changed("cidrs") {
				req.CIDRs = flags.cidrs
				changed = true
			}
			if cmd.Flags().Changed("schema") {
				req.Schema = &flags.schema
				changed = true
			}
			if cmd.Flags().Changed("role") {
				req.Role = &flags.role
				changed = true
			}
			if cmd.Flags().Changed("description") {
				req.Description = &flags.description
				changed = true
			}
			if !changed {
				return fmt.Errorf("at least one of --cidrs, --schema, --role, or --description must be provided")
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating IP restriction entry %s for %s", printer.BoldBlue(entryID), printer.BoldBlue(database)))
			defer end()

			entry, err := client.PostgresCIDRs.Update(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("IP restriction entry %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(entryID), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toIPRestrictionEntry(entry))
		},
	}

	cmd.Flags().StringSliceVar(&flags.cidrs, "cidrs", nil, "IPv4 CIDR ranges to allow (replaces the existing list)")
	cmd.Flags().StringVar(&flags.schema, "schema", "", "Postgres schema to restrict (empty for all schemas)")
	cmd.Flags().StringVar(&flags.role, "role", "", "Postgres role to restrict (empty for all roles)")
	cmd.Flags().StringVar(&flags.description, "description", "", "Description for the IP restriction entry (empty to clear)")

	return cmd
}
