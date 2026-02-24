package vtctld

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func MaterializeCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "materialize <command>",
		Short: "Manage Materialize workflows",
	}

	cmd.AddCommand(MaterializeCreateCmd(ch))
	cmd.AddCommand(MaterializeShowCmd(ch))
	cmd.AddCommand(MaterializeStartCmd(ch))
	cmd.AddCommand(MaterializeStopCmd(ch))
	cmd.AddCommand(MaterializeCancelCmd(ch))

	return cmd
}

func MaterializeCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow                     string
		targetKeyspace               string
		sourceKeyspace               string
		tableSettings                string
		cells                        []string
		referenceTables              []string
		tabletTypes                  []string
		stopAfterCopy                bool
		tabletTypesInPreferenceOrder bool
		deferSecondaryKeys           bool
		atomicCopy                   bool
		onDDL                        string
		sourceTimeZone               string
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a Materialize workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Creating Materialize workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.MaterializeCreateRequest{
				Organization:    ch.Config.Organization,
				Database:        database,
				Branch:          branch,
				Workflow:        flags.workflow,
				TargetKeyspace:  flags.targetKeyspace,
				SourceKeyspace:  flags.sourceKeyspace,
				TableSettings:   json.RawMessage(flags.tableSettings),
				Cells:           flags.cells,
				ReferenceTables: flags.referenceTables,
				TabletTypes:     flags.tabletTypes,
				OnDDL:           flags.onDDL,
				SourceTimeZone:  flags.sourceTimeZone,
			}

			if cmd.Flags().Changed("stop-after-copy") {
				req.StopAfterCopy = &flags.stopAfterCopy
			}
			if cmd.Flags().Changed("atomic-copy") {
				req.AtomicCopy = &flags.atomicCopy
			}
			if cmd.Flags().Changed("tablet-types-in-preference-order") {
				req.TabletTypesInPreferenceOrder = &flags.tabletTypesInPreferenceOrder
			}
			if cmd.Flags().Changed("defer-secondary-keys") {
				req.DeferSecondaryKeys = &flags.deferSecondaryKeys
			}

			data, err := client.Materialize.Create(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow (required)")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace (required)")
	cmd.Flags().StringVar(&flags.sourceKeyspace, "source-keyspace", "", "Source keyspace (required)")
	cmd.Flags().StringVar(&flags.tableSettings, "table-settings", "", "JSON array of table materialization settings (required)")
	cmd.Flags().StringSliceVar(&flags.cells, "cells", nil, "Cells to use")
	cmd.Flags().StringSliceVar(&flags.referenceTables, "reference-tables", nil, "Reference tables to include")
	cmd.Flags().StringSliceVar(&flags.tabletTypes, "tablet-types", nil, "Tablet types to use")
	cmd.Flags().BoolVar(&flags.stopAfterCopy, "stop-after-copy", false, "Stop the workflow after copying is complete")
	cmd.Flags().BoolVar(&flags.tabletTypesInPreferenceOrder, "tablet-types-in-preference-order", false, "Use tablet types in order of preference")
	cmd.Flags().BoolVar(&flags.deferSecondaryKeys, "defer-secondary-keys", false, "Defer secondary keys during copy")
	cmd.Flags().BoolVar(&flags.atomicCopy, "atomic-copy", false, "Use atomic copy")
	cmd.Flags().StringVar(&flags.onDDL, "on-ddl", "", "DDL handling strategy (IGNORE, STOP, EXEC, EXEC_IGNORE)")
	cmd.Flags().StringVar(&flags.sourceTimeZone, "source-time-zone", "", "Source time zone")

	_ = cmd.MarkFlagRequired("workflow")
	_ = cmd.MarkFlagRequired("target-keyspace")
	_ = cmd.MarkFlagRequired("source-keyspace")
	_ = cmd.MarkFlagRequired("table-settings")

	return cmd
}

func MaterializeShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow       string
		targetKeyspace string
		includeLogs    bool
	}

	cmd := &cobra.Command{
		Use:   "show <database> <branch>",
		Short: "Show a Materialize workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Fetching Materialize workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.MaterializeShowRequest{
				Organization:   ch.Config.Organization,
				Database:       database,
				Branch:         branch,
				Workflow:       flags.workflow,
				TargetKeyspace: flags.targetKeyspace,
			}

			if cmd.Flags().Changed("include-logs") {
				req.IncludeLogs = &flags.includeLogs
			}

			data, err := client.Materialize.Show(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow (required)")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace (required)")
	cmd.Flags().BoolVar(&flags.includeLogs, "include-logs", false, "Include workflow logs in the response")

	_ = cmd.MarkFlagRequired("workflow")
	_ = cmd.MarkFlagRequired("target-keyspace")

	return cmd
}

func MaterializeStartCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow       string
		targetKeyspace string
	}

	cmd := &cobra.Command{
		Use:   "start <database> <branch>",
		Short: "Start a Materialize workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Starting Materialize workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.Materialize.Start(ctx, &ps.MaterializeStartRequest{
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

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow (required)")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace (required)")

	_ = cmd.MarkFlagRequired("workflow")
	_ = cmd.MarkFlagRequired("target-keyspace")

	return cmd
}

func MaterializeStopCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow       string
		targetKeyspace string
	}

	cmd := &cobra.Command{
		Use:   "stop <database> <branch>",
		Short: "Stop a Materialize workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Stopping Materialize workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.Materialize.Stop(ctx, &ps.MaterializeStopRequest{
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

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow (required)")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace (required)")

	_ = cmd.MarkFlagRequired("workflow")
	_ = cmd.MarkFlagRequired("target-keyspace")

	return cmd
}

func MaterializeCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		workflow         string
		targetKeyspace   string
		keepData         bool
		keepRoutingRules bool
	}

	cmd := &cobra.Command{
		Use:   "cancel <database> <branch>",
		Short: "Cancel a Materialize workflow",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Canceling Materialize workflow %s on %s/%s\u2026",
					printer.BoldBlue(flags.workflow), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.MaterializeCancelRequest{
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

			data, err := client.Materialize.Cancel(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.workflow, "workflow", "", "Name of the workflow (required)")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace (required)")
	cmd.Flags().BoolVar(&flags.keepData, "keep-data", false, "Keep the data after canceling")
	cmd.Flags().BoolVar(&flags.keepRoutingRules, "keep-routing-rules", false, "Keep the routing rules after canceling")

	_ = cmd.MarkFlagRequired("workflow")
	_ = cmd.MarkFlagRequired("target-keyspace")

	return cmd
}
