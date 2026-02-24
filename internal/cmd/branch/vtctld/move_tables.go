package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func MoveTablesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move-tables <command>",
		Short: "Manage MoveTables workflows",
	}

	cmd.AddCommand(MoveTablesCreateCmd(ch))
	cmd.AddCommand(MoveTablesShowCmd(ch))
	cmd.AddCommand(MoveTablesStatusCmd(ch))
	cmd.AddCommand(MoveTablesSwitchTrafficCmd(ch))
	cmd.AddCommand(MoveTablesReverseTrafficCmd(ch))
	cmd.AddCommand(MoveTablesCancelCmd(ch))
	cmd.AddCommand(MoveTablesCompleteCmd(ch))

	return cmd
}

func MoveTablesCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow                     string
		targetKeyspace               string
		sourceKeyspace               string
		tables                       []string
		allTables                    bool
		autoStart                    bool
		stopAfterCopy                bool
		deferSecondaryKeys           bool
		onDDL                        string
		shardedAutoIncrementHandling string
		sourceTimeZone               string
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a MoveTables workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Creating MoveTables workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.MoveTablesCreateRequest{
				Organization:                 ch.Config.Organization,
				Database:                     database,
				Branch:                       branch,
				Workflow:                     flags.workflow,
				TargetKeyspace:               flags.targetKeyspace,
				SourceKeyspace:               flags.sourceKeyspace,
				Tables:                       flags.tables,
				DeferSecondaryKeys:           &flags.deferSecondaryKeys,
				OnDDL:                        flags.onDDL,
				ShardedAutoIncrementHandling: flags.shardedAutoIncrementHandling,
				SourceTimeZone:               flags.sourceTimeZone,
			}

			if cmd.Flags().Changed("all-tables") {
				req.AllTables = &flags.allTables
			}
			if cmd.Flags().Changed("stop-after-copy") {
				req.StopAfterCopy = &flags.stopAfterCopy
			}
			if cmd.Flags().Changed("auto-start") {
				req.AutoStart = &flags.autoStart
			}

			data, err := client.MoveTables.Create(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.Flags().StringVar(&flags.sourceKeyspace, "source-keyspace", "", "Source keyspace")
	cmd.Flags().StringSliceVar(&flags.tables, "tables", nil, "Tables to move (comma-separated)")
	cmd.Flags().BoolVar(&flags.allTables, "all-tables", false, "Move all tables from the source keyspace")
	cmd.Flags().BoolVar(&flags.autoStart, "auto-start", true, "Automatically start the workflow after creation")
	cmd.Flags().BoolVar(&flags.stopAfterCopy, "stop-after-copy", false, "Stop the workflow after the copy phase")
	cmd.Flags().BoolVar(&flags.deferSecondaryKeys, "defer-secondary-keys", true, "Defer secondary keys during the copy phase")
	cmd.Flags().StringVar(&flags.onDDL, "on-ddl", "", "DDL handling strategy (IGNORE, STOP, EXEC, EXEC_IGNORE)")
	cmd.Flags().StringVar(&flags.shardedAutoIncrementHandling, "sharded-auto-increment-handling", "", "Auto increment handling for sharded keyspaces")
	cmd.Flags().StringVar(&flags.sourceTimeZone, "source-time-zone", "", "Source time zone")

	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck
	cmd.MarkFlagRequired("source-keyspace") // nolint:errcheck
	cmd.MarkFlagsMutuallyExclusive("tables", "all-tables")

	return cmd
}

func MoveTablesShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow       string
		targetKeyspace string
	}

	cmd := &cobra.Command{
		Use:   "show <database> <branch>",
		Short: "Show details of a MoveTables workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Fetching MoveTables workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.MoveTables.Show(ctx, &ps.MoveTablesShowRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Workflow:       flags.workflow,
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
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck

	return cmd
}

func MoveTablesStatusCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow       string
		targetKeyspace string
	}

	cmd := &cobra.Command{
		Use:   "status <database> <branch>",
		Short: "Show the status of a MoveTables workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Fetching MoveTables workflow status for %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.MoveTables.Status(ctx, &ps.MoveTablesStatusRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Workflow:       flags.workflow,
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
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck

	return cmd
}

func MoveTablesSwitchTrafficCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow                  string
		targetKeyspace            string
		tabletTypes               []string
		dryRun                    bool
		initializeTargetSequences bool
	}

	cmd := &cobra.Command{
		Use:   "switch-traffic <database> <branch>",
		Short: "Switch traffic for a MoveTables workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Switching traffic for MoveTables workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.MoveTablesSwitchTrafficRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Workflow:       flags.workflow,
				TargetKeyspace: flags.targetKeyspace,
				TabletTypes:    flags.tabletTypes,
			}

			if cmd.Flags().Changed("dry-run") {
				req.DryRun = &flags.dryRun
			}
			if cmd.Flags().Changed("initialize-target-sequences") {
				req.InitializeTargetSequences = &flags.initializeTargetSequences
			}

			data, err := client.MoveTables.SwitchTraffic(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.Flags().StringSliceVar(&flags.tabletTypes, "tablet-types", nil, "Tablet types to switch traffic for (comma-separated)")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Only show what would be done")
	cmd.Flags().BoolVar(&flags.initializeTargetSequences, "initialize-target-sequences", false, "Initialize target sequences")
	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck

	return cmd
}

func MoveTablesReverseTrafficCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow       string
		targetKeyspace string
		dryRun         bool
	}

	cmd := &cobra.Command{
		Use:   "reverse-traffic <database> <branch>",
		Short: "Reverse traffic for a MoveTables workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Reversing traffic for MoveTables workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.MoveTablesReverseTrafficRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Workflow:       flags.workflow,
				TargetKeyspace: flags.targetKeyspace,
			}

			if cmd.Flags().Changed("dry-run") {
				req.DryRun = &flags.dryRun
			}

			data, err := client.MoveTables.ReverseTraffic(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Only show what would be done")
	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck

	return cmd
}

func MoveTablesCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow         string
		targetKeyspace   string
		keepData         bool
		keepRoutingRules bool
	}

	cmd := &cobra.Command{
		Use:   "cancel <database> <branch>",
		Short: "Cancel a MoveTables workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Canceling MoveTables workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.MoveTablesCancelRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Workflow:       flags.workflow,
				TargetKeyspace: flags.targetKeyspace,
			}

			if cmd.Flags().Changed("keep-data") {
				req.KeepData = &flags.keepData
			}
			if cmd.Flags().Changed("keep-routing-rules") {
				req.KeepRoutingRules = &flags.keepRoutingRules
			}

			data, err := client.MoveTables.Cancel(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.Flags().BoolVar(&flags.keepData, "keep-data", false, "Keep the data in the target keyspace")
	cmd.Flags().BoolVar(&flags.keepRoutingRules, "keep-routing-rules", false, "Keep the routing rules")
	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck

	return cmd
}

func MoveTablesCompleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow         string
		targetKeyspace   string
		keepData         bool
		keepRoutingRules bool
		dryRun           bool
	}

	cmd := &cobra.Command{
		Use:   "complete <database> <branch>",
		Short: "Complete a MoveTables workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Completing MoveTables workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.MoveTablesCompleteRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Workflow:       flags.workflow,
				TargetKeyspace: flags.targetKeyspace,
			}

			if cmd.Flags().Changed("keep-data") {
				req.KeepData = &flags.keepData
			}
			if cmd.Flags().Changed("keep-routing-rules") {
				req.KeepRoutingRules = &flags.keepRoutingRules
			}
			if cmd.Flags().Changed("dry-run") {
				req.DryRun = &flags.dryRun
			}

			data, err := client.MoveTables.Complete(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace")
	cmd.Flags().BoolVar(&flags.keepData, "keep-data", false, "Keep the data in the target keyspace")
	cmd.Flags().BoolVar(&flags.keepRoutingRules, "keep-routing-rules", false, "Keep the routing rules")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "Only show what would be done")
	cmd.MarkFlagRequired("workflow")        // nolint:errcheck
	cmd.MarkFlagRequired("target-keyspace") // nolint:errcheck

	return cmd
}
