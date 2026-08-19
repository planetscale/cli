package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// MaintenanceCmd groups the maintenance commands for a Postgres branch.
func MaintenanceCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "maintenance <command>",
		Short: "Run maintenance for a Postgres branch",
		Long: `Manage maintenance for a Postgres branch.

PlanetScale upgrades a branch's image in emergencies, such as patching security
issues, or when you initiate the upgrade yourself. 'maintenance run' initiates
one.

See https://planetscale.com/docs/postgres/operations-philosophy`,
	}

	cmd.AddCommand(MaintenanceRunCmd(ch))

	return cmd
}

// MaintenanceRunCmd starts a maintenance run for a Postgres branch.
func MaintenanceRunCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		updatePostgresMinorVersion bool
	}

	cmd := &cobra.Command{
		Use:   "run <database> <branch>",
		Short: "Run maintenance for a Postgres branch now (Postgres only)",
		Long: `Run maintenance for a Postgres branch, updating it to the latest available
image. Postgres only.

This is how regular version bumps, bugfixes, and quality-of-life improvements
reach a branch. PlanetScale otherwise upgrades images only in emergencies, such
as patching security issues.

The upgrade is applied to the replicas first, followed by a switchover from the
old primary to an upgraded replica. That failover leads to a short period of
database unavailability (seconds), and all direct connections are terminated, so
your application should have retry logic. A branch running a single instance has
no replica to switch over to and is unavailable until it comes back.

Pass --update-postgres-minor-version to also upgrade the branch to the latest
PostgreSQL minor version during the run.

Maintenance cannot start while a change request (see 'branch resize') is still
in progress.

See https://planetscale.com/docs/postgres/operations-philosophy`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := cmdutil.RequirePostgresDatabase(ctx, client, ch.Config.Organization, database, "Maintenance runs"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Starting maintenance for branch %s in %s...",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			err = client.BranchMaintenance.Run(ctx, &ps.RunBranchMaintenanceRequest{
				Organization:               ch.Config.Organization,
				Database:                   database,
				Branch:                     branch,
				UpdatePostgresMinorVersion: flags.updatePostgresMinorVersion,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return cmdutil.HandleNotFoundWithServiceTokenCheck(
						ctx, cmd, ch.Config, ch.Client, err, "write_database",
						"branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Maintenance for branch %s in %s was started.\n",
					printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result": "maintenance started",
				"branch": branch,
			})
		},
	}

	cmd.Flags().BoolVar(&flags.updatePostgresMinorVersion, "update-postgres-minor-version", false,
		"Upgrade the branch to the latest PostgreSQL minor version during maintenance")

	return cmd
}
