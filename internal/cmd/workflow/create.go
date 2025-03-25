package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

type createFlags struct {
	name               string
	sourceKeyspace     string
	targetKeyspace     string
	globalKeyspace     string
	tables             []string
	deferSecondaryKeys bool
	onDDL              string
	interactive        bool
}

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags createFlags

	cmd := &cobra.Command{
		Use:   "create <database> <branch>",
		Short: "Create a new workflow within a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			org := ch.Config.Organization

			if flags.interactive {
				return createInteractive(ctx, ch, org, db, branch, flags)
			}

			end := ch.Printer.PrintProgress("Creating workflow…")
			defer end()

			workflow, err := createWorkflow(ctx, client, org, db, branch, flags)
			end()
			
			if err != nil {
				return err
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Successfully created workflow %s to copy %d tables from %s to %s.\n", printer.BoldBlue(workflow.Name), len(flags.tables), printer.Bold(flags.sourceKeyspace), printer.Bold(flags.targetKeyspace))
				return nil
			}

			return ch.Printer.PrintResource(toWorkflow(workflow))
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.sourceKeyspace, "source-keyspace", "", "Keyspace where the tables will be copied from.")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Keyspace where the tables will be copied to.")
	cmd.Flags().StringVar(&flags.globalKeyspace, "global-keyspace", "", "Choose an unsharded keyspace where sequence tables will be created for any workflow table that contains `AUTO_INCREMENT`")
	cmd.Flags().StringSliceVar(&flags.tables, "tables", []string{}, "Tables to migrate to the target keyspace")
	cmd.Flags().BoolVar(&flags.deferSecondaryKeys, "defer-secondary-keys", true, "Don't create secondary indexes for tables until they've been copied")
	cmd.Flags().StringVar(&flags.onDDL, "on-ddl", "STOP", "Action to take when a DDL statement is encountered during a running workflow. Options: EXEC, EXEC_IGNORE, STOP, IGNORE")
	cmd.Flags().BoolVarP(&flags.interactive, "interactive", "i", false, "Create the workflow in interactive mode")

	cmd.RegisterFlagCompletionFunc("on-ddl", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			cobra.CompletionWithDesc("EXEC", "Try to execute the DDL on the target keyspace. If it fails, stop the workflow."),
			cobra.CompletionWithDesc("EXEC_IGNORE", "Try to execute the DDL on the target keyspace, ignoring any errors."),
			cobra.CompletionWithDesc("STOP", "Stop the workflow on DDL."),
			cobra.CompletionWithDesc("IGNORE", "Ignore the DDL statement and continue."),
		}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func createWorkflow(ctx context.Context, client *ps.Client, org, db, branch string, flags createFlags) (*ps.Workflow, error) {
	workflow, err := client.Workflows.Create(ctx, &ps.CreateWorkflowRequest{
		Organization:       org,
		Database:           db,
		Branch:             branch,
		Name:               flags.name,
		SourceKeyspace:     flags.sourceKeyspace,
		TargetKeyspace:     flags.targetKeyspace,
		GlobalKeyspace:     &flags.globalKeyspace,
		Tables:             flags.tables,
		DeferSecondaryKeys: &flags.deferSecondaryKeys,
		OnDDL:              &flags.onDDL,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case ps.ErrNotFound:
			return nil, fmt.Errorf("keyspace %s, %s, or %s does not exist in branch %s (database: %s, organization: %s)", printer.BoldBlue(flags.sourceKeyspace), printer.BoldBlue(flags.targetKeyspace), printer.BoldBlue(flags.globalKeyspace), printer.BoldBlue(branch), printer.BoldBlue(db), printer.BoldBlue(org))
		default:
			return nil, cmdutil.HandleError(err)
		}
	}

	return workflow, nil
}

func createInteractive(ctx context.Context, ch *cmdutil.Helper, org, db, branch string, flags createFlags) error {
	client, err := ch.Client()
	if err != nil {
		return err
	}

	keyspaces, err := client.Keyspaces.List(ctx, &ps.ListKeyspacesRequest{
		Organization: org,
		Database:     db,
		Branch:       branch,
	})
	if err != nil {
		return cmdutil.HandleError(err)
	}

	form := huh.NewForm(
		// Naming the workflow
		huh.NewGroup(
			huh.NewInput().
				Title("Enter a name for this workflow").
				Value(&flags.name),
		),

		// Source keyspace and tables
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a source keyspace").
				Description("Choose the keyspace where the tables you want to copy are located").
				Value(&flags.sourceKeyspace).
				Validate(func(value string) error {
					if value == flags.targetKeyspace {
						return fmt.Errorf("source keyspace and target keyspace cannot be the same")
					}

					return nil
				}).
				OptionsFunc(func() []huh.Option[string] {
					keyspaceStrs := make([]string, 0, len(keyspaces))
					for _, keyspace := range keyspaces {
						if keyspace.Name != flags.targetKeyspace {
							keyspaceStrs = append(keyspaceStrs, keyspace.Name)
						}
					}

					return huh.NewOptions(keyspaceStrs...)
				}, &flags.targetKeyspace),
		),

		// Select tables
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select tables to replicate").
				Description("Choose the tables you want to copy to the target keyspace").
				Value(&flags.tables).
				Validate(func(values []string) error {
					if len(values) == 0 {
						return fmt.Errorf("please select at least one table")
					}

					return nil
				}).
				OptionsFunc(func() []huh.Option[string] {
					tables, err := client.DatabaseBranches.Schema(ctx, &ps.BranchSchemaRequest{
						Organization: org,
						Database:     db,
						Branch:       branch,
						Keyspace:     flags.sourceKeyspace,
					})
					if err != nil {
						return nil
					}

					tableStrs := make([]string, 0, len(tables))
					for _, table := range tables {
						if !strings.HasSuffix(table.Name, "_seq") {
							tableStrs = append(tableStrs, table.Name)
						}
					}

					return huh.NewOptions(tableStrs...)
				}, &flags.sourceKeyspace),
		),

		// Target keyspace
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a target keyspace").
				Description("Choose the keyspace where the tables will be copied to").
				Value(&flags.targetKeyspace).
				OptionsFunc(func() []huh.Option[string] {
					keyspaceStrs := make([]string, 0, len(keyspaces))
					for _, keyspace := range keyspaces {
						if keyspace.Name != flags.sourceKeyspace {
							keyspaceStrs = append(keyspaceStrs, keyspace.Name)
						}
					}

					return huh.NewOptions(keyspaceStrs...)
				}, &flags.sourceKeyspace),
		),

		// Advanced options
		huh.NewGroup(
			huh.NewConfirm().
				Title("Defer secondary index creation?").
				Description("Don't create secondary indexes for tables until they've been copied").
				Value(&flags.deferSecondaryKeys),
		).Title("Advanced options"),

		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select DDL Handling (--on-ddl)").
				Description("Action to take when a DDL statement is encountered during a running workflow").
				Value(&flags.onDDL).
				Options(
					huh.NewOption("EXEC - Try to execute the DDL on the target keyspace. If it fails, stop the workflow", "EXEC"),
					huh.NewOption("EXEC_IGNORE - Try to execute the DDL on the target keyspace, ignoring any errors", "EXEC_IGNORE"),
					huh.NewOption("STOP - Stop the workflow on DDL", "STOP"),
					huh.NewOption("IGNORE - Ignore any DDL and continue", "IGNORE"),
				),
		).Title("Advanced options"),

		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a global keyspace").
				Description("Choose an unsharded keyspace where sequence tables will be created for any workflow table that contains `AUTO_INCREMENT`").
				Value(&flags.globalKeyspace).
				Validate(func(value string) error {
					if value == flags.targetKeyspace {
						return fmt.Errorf("global keyspace and target keyspace cannot be the same")
					}

					globalKeyspace := findKeyspace(keyspaces, value)
					if globalKeyspace != nil && globalKeyspace.Sharded {
						return fmt.Errorf("global keyspace %s must be unsharded", value)
					}

					sourceKeyspace := findKeyspace(keyspaces, flags.sourceKeyspace)
					if sourceKeyspace != nil && sourceKeyspace.Sharded {
						return fmt.Errorf("source keyspace must be unsharded")
					}

					targetKeyspace := findKeyspace(keyspaces, flags.targetKeyspace)
					if targetKeyspace != nil && !targetKeyspace.Sharded {
						return fmt.Errorf("target keyspace must be sharded")
					}

					return nil
				}).
				OptionsFunc(func() []huh.Option[string] {
					keyspaceStrs := make([]string, 0, len(keyspaces))

					for _, keyspace := range keyspaces {
						if !keyspace.Sharded && keyspace.Name != flags.targetKeyspace {
							keyspaceStrs = append(keyspaceStrs, keyspace.Name)
						}
					}

					return huh.NewOptions(keyspaceStrs...)
				}, &flags.targetKeyspace),
		).
			WithHideFunc(func() bool {
				sourceKeyspace := findKeyspace(keyspaces, flags.sourceKeyspace)
				targetKeyspace := findKeyspace(keyspaces, flags.targetKeyspace)

				return sourceKeyspace == nil || targetKeyspace == nil || sourceKeyspace.Sharded || !targetKeyspace.Sharded
			}).
			Title("Advanced options"),
	).WithTheme(huh.ThemeBase16())

	err = form.Run()
	if err != nil {
		return err
	}

	workflow, err := createWorkflow(ctx, client, org, db, branch, flags)
	if err != nil {
		return err
	}

	ch.Printer.Printf("Successfully created workflow %s. It will copy %s tables from %s to %s.", printer.BoldBlue(workflow.Name), printer.Bold(len(workflow.Tables)), printer.BoldBlue(workflow.SourceKeyspace.Name), printer.BoldBlue(workflow.TargetKeyspace.Name))

	return nil
}

func findKeyspace(keyspaces []*ps.Keyspace, name string) *ps.Keyspace {
	for _, keyspace := range keyspaces {
		if keyspace.Name == name {
			return keyspace
		}
	}

	return nil
}