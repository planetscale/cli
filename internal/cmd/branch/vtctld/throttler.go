package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ThrottlerCmd groups the tablet throttler subcommands. The throttler controls
// how aggressively Vitess applies background work (such as Online DDL and
// VReplication) based on replication lag and other metrics.
func ThrottlerCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "throttler <command>",
		Short: "Inspect and configure the tablet throttler",
	}

	cmd.AddCommand(ThrottlerStatusCmd(ch))
	cmd.AddCommand(ThrottlerCheckCmd(ch))
	cmd.AddCommand(ThrottlerUpdateConfigCmd(ch))

	return cmd
}

// ThrottlerStatusCmd reads the live throttler status from a single tablet.
func ThrottlerStatusCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		tabletAlias string
	}

	cmd := &cobra.Command{
		Use:   "status <database> <branch>",
		Short: "Get the throttler status for a single tablet",
		Long: "Get the throttler status for a single tablet, identified by its alias. " +
			"Discover tablet aliases with `pscale branch vtctld list-tablets`.",
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Fetching throttler status for tablet %s on %s…",
					printer.BoldBlue(flags.tabletAlias), progressTarget(ch.Config.Organization, database, branch)))
			defer end()

			data, err := client.Vtctld.GetThrottlerStatus(ctx, &ps.VtctldGetThrottlerStatusRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				TabletAlias:  flags.tabletAlias,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.tabletAlias, "tablet-alias", "", "Alias of the tablet to probe (e.g. \"zone1-0000000100\")")
	cmd.MarkFlagRequired("tablet-alias") // nolint:errcheck

	return cmd
}

// ThrottlerCheckCmd issues a throttler check against a single tablet.
func ThrottlerCheckCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		tabletAlias           string
		appName               string
		scope                 string
		skipRequestHeartbeats bool
		okIfNotExists         bool
	}

	cmd := &cobra.Command{
		Use:   "check <database> <branch>",
		Short: "Issue a throttler check against a single tablet",
		Long: "Issue a throttler check against a single tablet, identified by its alias. " +
			"Discover tablet aliases with `pscale branch vtctld list-tablets`.",
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Checking throttler for tablet %s on %s…",
					printer.BoldBlue(flags.tabletAlias), progressTarget(ch.Config.Organization, database, branch)))
			defer end()

			req := &ps.VtctldCheckThrottlerRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				TabletAlias:  flags.tabletAlias,
				AppName:      flags.appName,
				Scope:        flags.scope,
			}
			if cmd.Flags().Changed("skip-request-heartbeats") {
				req.SkipRequestHeartbeats = &flags.skipRequestHeartbeats
			}
			if cmd.Flags().Changed("ok-if-not-exists") {
				req.OkIfNotExists = &flags.okIfNotExists
			}

			data, err := client.Vtctld.CheckThrottler(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.tabletAlias, "tablet-alias", "", "Alias of the tablet to check (e.g. \"zone1-0000000100\")")
	cmd.Flags().StringVar(&flags.appName, "app-name", "", "App to issue the check on behalf of (e.g. \"online-ddl\"). Defaults to the throttler's default app.")
	cmd.Flags().StringVar(&flags.scope, "scope", "", "Scope of the check, either \"shard\" or \"self\". Defaults to the throttler's default scope.")
	cmd.Flags().BoolVar(&flags.skipRequestHeartbeats, "skip-request-heartbeats", false, "Do not renew the throttler's heartbeat lease while serving this check")
	cmd.Flags().BoolVar(&flags.okIfNotExists, "ok-if-not-exists", false, "Return OK even if the requested metric does not exist")
	cmd.MarkFlagRequired("tablet-alias") // nolint:errcheck

	return cmd
}

// ThrottlerUpdateConfigCmd updates the tablet throttler configuration for a
// keyspace.
func ThrottlerUpdateConfigCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace  string
		enabled   bool
		threshold float64
	}

	cmd := &cobra.Command{
		Use:   "update-config <database> <branch>",
		Short: "Update the throttler configuration for a keyspace",
		Long: "Update the tablet throttler configuration for a keyspace. The throttler is " +
			"enabled or disabled with --enabled; this flag is required because there is no " +
			"separate \"leave unchanged\" state.",
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Updating throttler config for keyspace %s on %s…",
					printer.BoldBlue(flags.keyspace), progressTarget(ch.Config.Organization, database, branch)))
			defer end()

			req := &ps.VtctldUpdateThrottlerConfigRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     flags.keyspace,
				Enabled:      flags.enabled,
			}
			if cmd.Flags().Changed("threshold") {
				req.Threshold = &flags.threshold
			}

			data, err := client.Vtctld.UpdateThrottlerConfig(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace whose throttler config to update")
	cmd.Flags().BoolVar(&flags.enabled, "enabled", false, "Enable (true) or disable (false) the throttler for the keyspace")
	cmd.Flags().Float64Var(&flags.threshold, "threshold", 0, "Replication lag threshold in seconds for the default check (defaults to 5.0 server-side when omitted)")
	cmd.MarkFlagRequired("keyspace") // nolint:errcheck
	cmd.MarkFlagRequired("enabled")  // nolint:errcheck

	return cmd
}
