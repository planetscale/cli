package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func VDiffCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vdiff <command>",
		Short: "Manage VDiff operations",
	}

	cmd.AddCommand(VDiffCreateCmd(ch))
	cmd.AddCommand(VDiffShowCmd(ch))
	cmd.AddCommand(VDiffStopCmd(ch))
	cmd.AddCommand(VDiffResumeCmd(ch))
	cmd.AddCommand(VDiffDeleteCmd(ch))

	return cmd
}

func VDiffCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow                    string
		targetKeyspace              string
		autoRetry                   bool
		autoStart                   bool
		debugQuery                  bool
		onlyPKs                     bool
		updateTableStats            bool
		verbose                     bool
		tables                      []string
		tabletTypes                 []string
		tabletSelectionPreference   string
		filteredReplicationWaitTime int
		maxReportSampleRows         int
		maxExtraRowsToCompare       int
		rowDiffColumnTruncateAt     int
		limit                       int
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a VDiff",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Creating VDiff for workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.VDiffCreateRequest{
				Organization:              ch.Config.Organization,
				Database:                  database,
				Branch:                    branch,
				Workflow:                  flags.workflow,
				TargetKeyspace:            flags.targetKeyspace,
				DebugQuery:                flags.debugQuery,
				OnlyPKs:                   flags.onlyPKs,
				UpdateTableStats:          flags.updateTableStats,
				Verbose:                   flags.verbose,
				Tables:                    flags.tables,
				TabletTypes:               flags.tabletTypes,
				TabletSelectionPreference: flags.tabletSelectionPreference,
			}

			if cmd.Flags().Changed("auto-retry") {
				req.AutoRetry = &flags.autoRetry
			}
			if cmd.Flags().Changed("auto-start") {
				req.AutoStart = &flags.autoStart
			}
			if cmd.Flags().Changed("filtered-replication-wait-time") {
				req.FilteredReplicationWaitTime = &flags.filteredReplicationWaitTime
			}
			if cmd.Flags().Changed("max-report-sample-rows") {
				req.MaxReportSampleRows = &flags.maxReportSampleRows
			}
			if cmd.Flags().Changed("max-extra-rows-to-compare") {
				req.MaxExtraRowsToCompare = &flags.maxExtraRowsToCompare
			}
			if cmd.Flags().Changed("row-diff-column-truncate-at") {
				req.RowDiffColumnTruncateAt = &flags.rowDiffColumnTruncateAt
			}
			if cmd.Flags().Changed("limit") {
				req.Limit = &flags.limit
			}

			data, err := client.VDiff.Create(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.Flags().BoolVar(&flags.autoRetry, "auto-retry", true, "Automatically retry on error")
	cmd.Flags().BoolVar(&flags.autoStart, "auto-start", true, "Automatically start the VDiff")
	cmd.Flags().BoolVar(&flags.debugQuery, "debug-query", false, "Log the queries used for the VDiff")
	cmd.Flags().BoolVar(&flags.onlyPKs, "only-pks", false, "Only compare primary keys")
	cmd.Flags().BoolVar(&flags.updateTableStats, "update-table-stats", false, "Update table statistics before the VDiff")
	cmd.Flags().BoolVar(&flags.verbose, "verbose", false, "Verbose output")
	cmd.Flags().StringSliceVar(&flags.tables, "tables", nil, "Tables to compare (comma-separated)")
	cmd.Flags().StringSliceVar(&flags.tabletTypes, "tablet-types", nil, "Tablet types to use (comma-separated)")
	cmd.Flags().StringVar(&flags.tabletSelectionPreference, "tablet-selection-preference", "", "Tablet selection preference")
	cmd.Flags().IntVar(&flags.filteredReplicationWaitTime, "filtered-replication-wait-time", 0, "Filtered replication wait time in seconds")
	cmd.Flags().IntVar(&flags.maxReportSampleRows, "max-report-sample-rows", 0, "Maximum number of sample rows in report")
	cmd.Flags().IntVar(&flags.maxExtraRowsToCompare, "max-extra-rows-to-compare", 0, "Maximum extra rows to compare")
	cmd.Flags().IntVar(&flags.rowDiffColumnTruncateAt, "row-diff-column-truncate-at", 0, "Truncate column values at this length in the report")
	cmd.Flags().IntVar(&flags.limit, "limit", 0, "Maximum number of rows to compare")
	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck

	return cmd
}

func VDiffShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow       string
		uuid           string
		targetKeyspace string
	}

	cmd := &cobra.Command{
		Use:   "show <database> <branch>",
		Short: "Show details of a VDiff",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Fetching VDiff %s on %s/%s\u2026",
					printer.BoldBlue(flags.uuid), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.VDiff.Show(ctx, &ps.VDiffShowRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Workflow:       flags.workflow,
				UUID:           flags.uuid,
				TargetKeyspace: flags.targetKeyspace,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.uuid, "uuid", "", "UUID of the VDiff")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("uuid")            // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck

	return cmd
}

func VDiffStopCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow       string
		uuid           string
		targetKeyspace string
		targetShards   []string
	}

	cmd := &cobra.Command{
		Use:   "stop <database> <branch>",
		Short: "Stop a VDiff",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Stopping VDiff %s on %s/%s\u2026",
					printer.BoldBlue(flags.uuid), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.VDiff.Stop(ctx, &ps.VDiffStopRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Workflow:       flags.workflow,
				UUID:           flags.uuid,
				TargetKeyspace: flags.targetKeyspace,
				TargetShards:   flags.targetShards,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.uuid, "uuid", "", "UUID of the VDiff")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.Flags().StringSliceVar(&flags.targetShards, "target-shards", nil, "Target shards to stop (comma-separated)")
	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("uuid")            // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck

	return cmd
}

func VDiffResumeCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow       string
		uuid           string
		targetKeyspace string
		targetShards   []string
	}

	cmd := &cobra.Command{
		Use:   "resume <database> <branch>",
		Short: "Resume a stopped VDiff",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Resuming VDiff %s on %s/%s\u2026",
					printer.BoldBlue(flags.uuid), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.VDiff.Resume(ctx, &ps.VDiffResumeRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Workflow:       flags.workflow,
				UUID:           flags.uuid,
				TargetKeyspace: flags.targetKeyspace,
				TargetShards:   flags.targetShards,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.uuid, "uuid", "", "UUID of the VDiff")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.Flags().StringSliceVar(&flags.targetShards, "target-shards", nil, "Target shards to resume (comma-separated)")
	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("uuid")            // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck

	return cmd
}

func VDiffDeleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow       string
		uuid           string
		targetKeyspace string
	}

	cmd := &cobra.Command{
		Use:   "delete <database> <branch>",
		Short: "Delete a VDiff",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Deleting VDiff %s on %s/%s\u2026",
					printer.BoldBlue(flags.uuid), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.VDiff.Delete(ctx, &ps.VDiffDeleteRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Workflow:       flags.workflow,
				UUID:           flags.uuid,
				TargetKeyspace: flags.targetKeyspace,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.uuid, "uuid", "", "UUID of the VDiff")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("uuid")            // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck

	return cmd
}
