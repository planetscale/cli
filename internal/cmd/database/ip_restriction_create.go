package database

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// IPRestrictionCreateCmd creates an IP restriction entry.
func IPRestrictionCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		cidrs       []string
		schema      string
		role        string
		description string
	}

	cmd := &cobra.Command{
		Use:   "create <database>",
		Short: "Create an IP restriction entry",
		Long: `Create an IP restriction entry for a PostgreSQL database.

At least one IPv4 CIDR is required (e.g. 192.168.1.0/24 or 203.0.113.10/32).
Omit --schema / --role (or leave them empty) to apply the restriction to all
schemas and roles.`,
		Args: cmdutil.RequiredArgs("database"),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := requirePostgresDatabase(ctx, ch, client, database); err != nil {
				return err
			}

			req := &ps.CreatePostgresCIDRRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Schema:       flags.schema,
				Role:         flags.role,
				CIDRs:        flags.cidrs,
				Description:  flags.description,
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating IP restriction entry for %s", printer.BoldBlue(database)))
			defer end()

			entry, err := client.PostgresCIDRs.Create(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toIPRestrictionEntry(entry))
		},
	}

	cmd.Flags().StringSliceVar(&flags.cidrs, "cidrs", nil, "IPv4 CIDR ranges to allow (required, repeatable or comma-separated)")
	cmd.Flags().StringVar(&flags.schema, "schema", "", "Postgres schema to restrict (omit for all schemas)")
	cmd.Flags().StringVar(&flags.role, "role", "", "Postgres role to restrict (omit for all roles)")
	cmd.Flags().StringVar(&flags.description, "description", "", "Optional description for the IP restriction entry")

	cmd.MarkFlagRequired("cidrs") // nolint:errcheck

	return cmd
}
