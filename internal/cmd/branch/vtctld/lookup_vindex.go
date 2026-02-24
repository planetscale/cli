package vtctld

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func LookupVindexCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lookup-vindex <command>",
		Short: "Manage Lookup Vindex operations",
	}

	cmd.AddCommand(LookupVindexCreateCmd(ch))
	cmd.AddCommand(LookupVindexShowCmd(ch))
	cmd.AddCommand(LookupVindexExternalizeCmd(ch))
	cmd.AddCommand(LookupVindexInternalizeCmd(ch))
	cmd.AddCommand(LookupVindexCancelCmd(ch))
	cmd.AddCommand(LookupVindexCompleteCmd(ch))

	return cmd
}

func LookupVindexCreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name                         string
		tableKeyspace                string
		keyspace                     string
		vindexType                   string
		cells                        []string
		tabletTypes                  []string
		tableOwner                   string
		tableName                    string
		tableOwnerColumns            []string
		tableVindexType              string
		ignoreNulls                  bool
		tabletTypesInPreferenceOrder bool
		continueAfterCopyWithOwner   bool
	}

	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a Lookup Vindex",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Creating Lookup Vindex %s on %s/%s\u2026",
					printer.BoldBlue(flags.name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.LookupVindexCreateRequest{
				Organization:      ch.Config.Organization,
				Database:          database,
				Branch:            branch,
				Name:              flags.name,
				TableKeyspace:     flags.tableKeyspace,
				Keyspace:          flags.keyspace,
				Type:              flags.vindexType,
				Cells:             flags.cells,
				TabletTypes:       flags.tabletTypes,
				TableOwner:        flags.tableOwner,
				TableName:         flags.tableName,
				TableOwnerColumns: flags.tableOwnerColumns,
				TableVindexType:   flags.tableVindexType,
			}
			if cmd.Flags().Changed("ignore-nulls") {
				req.IgnoreNulls = &flags.ignoreNulls
			}
			if cmd.Flags().Changed("tablet-types-in-preference-order") {
				req.TabletTypesInPreferenceOrder = &flags.tabletTypesInPreferenceOrder
			}
			if cmd.Flags().Changed("continue-after-copy-with-owner") {
				req.ContinueAfterCopyWithOwner = &flags.continueAfterCopyWithOwner
			}

			data, err := client.LookupVindex.Create(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the Lookup Vindex")
	cmd.Flags().StringVar(&flags.tableKeyspace, "table-keyspace", "", "Keyspace of the table to create the vindex on")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace for the lookup table")
	cmd.Flags().StringVar(&flags.vindexType, "type", "", "Type of the vindex")
	cmd.Flags().StringSliceVar(&flags.cells, "cells", nil, "Cells to replicate from (comma-separated)")
	cmd.Flags().StringSliceVar(&flags.tabletTypes, "tablet-types", nil, "Tablet types to replicate from (comma-separated)")
	cmd.Flags().StringVar(&flags.tableOwner, "table-owner", "", "Owner table name")
	cmd.Flags().StringVar(&flags.tableName, "table-name", "", "Name of the lookup table")
	cmd.Flags().StringSliceVar(&flags.tableOwnerColumns, "table-owner-columns", nil, "Owner table columns (comma-separated)")
	cmd.Flags().StringVar(&flags.tableVindexType, "table-vindex-type", "", "Vindex type on the owner table")
	cmd.Flags().BoolVar(&flags.ignoreNulls, "ignore-nulls", false, "Ignore null values")
	cmd.Flags().BoolVar(&flags.tabletTypesInPreferenceOrder, "tablet-types-in-preference-order", false, "Use tablet types in preference order")
	cmd.Flags().BoolVar(&flags.continueAfterCopyWithOwner, "continue-after-copy-with-owner", false, "Continue after copy with owner")
	cmd.MarkFlagRequired("name")           // nolint:errcheck
	cmd.MarkFlagRequired("table-keyspace") // nolint:errcheck

	return cmd
}

func LookupVindexShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name          string
		tableKeyspace string
	}

	cmd := &cobra.Command{
		Use:   "show <database> <branch>",
		Short: "Show details of a Lookup Vindex",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Fetching Lookup Vindex %s on %s/%s\u2026",
					printer.BoldBlue(flags.name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.LookupVindex.Show(ctx, &ps.LookupVindexShowRequest{
				Organization:  ch.Config.Organization,
				Database:      database,
				Branch:        branch,
				Name:          flags.name,
				TableKeyspace: flags.tableKeyspace,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the Lookup Vindex")
	cmd.Flags().StringVar(&flags.tableKeyspace, "table-keyspace", "", "Keyspace of the table")
	cmd.MarkFlagRequired("name")           // nolint:errcheck
	cmd.MarkFlagRequired("table-keyspace") // nolint:errcheck

	return cmd
}

func LookupVindexExternalizeCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name          string
		tableKeyspace string
		keyspace      string
		delete        bool
	}

	cmd := &cobra.Command{
		Use:   "externalize <database> <branch>",
		Short: "Externalize a Lookup Vindex",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Externalizing Lookup Vindex %s on %s/%s\u2026",
					printer.BoldBlue(flags.name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			req := &ps.LookupVindexExternalizeRequest{
				Organization:  ch.Config.Organization,
				Database:      database,
				Branch:        branch,
				Name:          flags.name,
				TableKeyspace: flags.tableKeyspace,
				Keyspace:      flags.keyspace,
			}

			if cmd.Flags().Changed("delete") {
				req.Delete = &flags.delete
			}

			data, err := client.LookupVindex.Externalize(ctx, req)
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the Lookup Vindex")
	cmd.Flags().StringVar(&flags.tableKeyspace, "table-keyspace", "", "Keyspace of the table")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace for the lookup table")
	cmd.Flags().BoolVar(&flags.delete, "delete", false, "Delete the vindex after externalizing")
	cmd.MarkFlagRequired("name")           // nolint:errcheck
	cmd.MarkFlagRequired("table-keyspace") // nolint:errcheck

	return cmd
}

func LookupVindexInternalizeCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name          string
		tableKeyspace string
		keyspace      string
	}

	cmd := &cobra.Command{
		Use:   "internalize <database> <branch>",
		Short: "Internalize a Lookup Vindex",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Internalizing Lookup Vindex %s on %s/%s\u2026",
					printer.BoldBlue(flags.name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.LookupVindex.Internalize(ctx, &ps.LookupVindexInternalizeRequest{
				Organization:  ch.Config.Organization,
				Database:      database,
				Branch:        branch,
				Name:          flags.name,
				TableKeyspace: flags.tableKeyspace,
				Keyspace:      flags.keyspace,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the Lookup Vindex")
	cmd.Flags().StringVar(&flags.tableKeyspace, "table-keyspace", "", "Keyspace of the table")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace for the lookup table")
	cmd.MarkFlagRequired("name")           // nolint:errcheck
	cmd.MarkFlagRequired("table-keyspace") // nolint:errcheck

	return cmd
}

func LookupVindexCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name          string
		tableKeyspace string
	}

	cmd := &cobra.Command{
		Use:   "cancel <database> <branch>",
		Short: "Cancel a Lookup Vindex creation",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Canceling Lookup Vindex %s on %s/%s\u2026",
					printer.BoldBlue(flags.name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.LookupVindex.Cancel(ctx, &ps.LookupVindexCancelRequest{
				Organization:  ch.Config.Organization,
				Database:      database,
				Branch:        branch,
				Name:          flags.name,
				TableKeyspace: flags.tableKeyspace,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the Lookup Vindex")
	cmd.Flags().StringVar(&flags.tableKeyspace, "table-keyspace", "", "Keyspace of the table")
	cmd.MarkFlagRequired("name")           // nolint:errcheck
	cmd.MarkFlagRequired("table-keyspace") // nolint:errcheck

	return cmd
}

func LookupVindexCompleteCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name          string
		tableKeyspace string
		keyspace      string
	}

	cmd := &cobra.Command{
		Use:   "complete <database> <branch>",
		Short: "Complete a Lookup Vindex creation",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Completing Lookup Vindex %s on %s/%s\u2026",
					printer.BoldBlue(flags.name), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			data, err := client.LookupVindex.Complete(ctx, &ps.LookupVindexCompleteRequest{
				Organization:  ch.Config.Organization,
				Database:      database,
				Branch:        branch,
				Name:          flags.name,
				TableKeyspace: flags.tableKeyspace,
				Keyspace:      flags.keyspace,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the Lookup Vindex")
	cmd.Flags().StringVar(&flags.tableKeyspace, "table-keyspace", "", "Keyspace of the table")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Keyspace for the lookup table")
	cmd.MarkFlagRequired("name")           // nolint:errcheck
	cmd.MarkFlagRequired("table-keyspace") // nolint:errcheck

	return cmd
}
