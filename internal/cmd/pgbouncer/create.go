package pgbouncer

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// CreateCmd creates a dedicated PgBouncer.
func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name            string
		target          string
		size            string
		replicasPerCell int
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a dedicated PgBouncer",
		Long: `Create a dedicated PgBouncer for a PostgreSQL database branch.

Target must be one of: primary, replica, replica_az_affinity.
Size is a PgBouncer SKU name (for example PGB_10). If omitted, the API picks
a default size. Name is optional and auto-generated when omitted (max 12
characters; "primary" and "replica" are reserved).`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "PgBouncers"); err != nil {
				return err
			}

			req := &ps.CreatePostgresBouncerRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Name:         flags.name,
				Target:       flags.target,
				BouncerSize:  flags.size,
			}
			if cmd.Flags().Changed("replicas-per-cell") {
				req.ReplicasPerCell = &flags.replicasPerCell
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating PgBouncer for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			bouncer, err := client.PostgresBouncers.Create(ctx, req)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s or branch %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toPgBouncer(bouncer))
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name for the PgBouncer (optional; auto-generated if omitted)")
	cmd.Flags().StringVar(&flags.target, "target", "", "Traffic target: primary, replica, or replica_az_affinity (required)")
	cmd.Flags().StringVar(&flags.size, "size", "", "PgBouncer size SKU (e.g. PGB_10)")
	cmd.Flags().IntVar(&flags.replicasPerCell, "replicas-per-cell", 1, "Number of replica servers per cell")

	cmd.MarkFlagRequired("target") // nolint:errcheck

	return cmd
}
