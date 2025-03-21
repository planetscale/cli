package workflow

import (
	"context"
	"fmt"

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

			if flags.globalKeyspace == "" {
				flags.globalKeyspace = flags.sourceKeyspace
			}

			if flags.interactive {
				return createInteractive(ctx, client, org, db, branch, flags)
			}

			end := ch.Printer.PrintProgress("Creating workflow...")
			defer end()

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
					return fmt.Errorf("keyspace %s, %s, or %s does not exist in branch %s (database: %s, organization: %s)", printer.BoldBlue(flags.sourceKeyspace), printer.BoldBlue(flags.targetKeyspace), printer.BoldBlue(flags.globalKeyspace), printer.BoldBlue(branch), printer.BoldBlue(db), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Successfully created workflow %s to copy %d tables from %s to %s.\n", printer.BoldBlue(workflow.Name), len(flags.tables), printer.Bold(flags.sourceKeyspace), printer.Bold(flags.targetKeyspace))
				return nil
			}

			return ch.Printer.PrintResource(toWorkflow(workflow))
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the workflow")
	cmd.Flags().StringVar(&flags.sourceKeyspace, "source-keyspace", "", "Source keyspace for the workflow")
	cmd.Flags().StringVar(&flags.targetKeyspace, "target-keyspace", "", "Target keyspace for the workflow")
	cmd.Flags().StringVar(&flags.globalKeyspace, "global-keyspace", "", "Global keyspace for the workflow")
	cmd.Flags().StringSliceVar(&flags.tables, "tables", []string{}, "Tables to migrate to the target keyspace")
	cmd.Flags().BoolVar(&flags.deferSecondaryKeys, "defer-secondary-keys", true, "Don’t create secondary indexes for tables until they’ve been copied")
	cmd.Flags().StringVar(&flags.onDDL, "on-ddl", "STOP", "Action to take when a DDL statement is encountered during a running workflow. Options: EXEC, EXEC_IGNORE, STOP, IGNORE")
	cmd.Flags().BoolVarP(&flags.interactive, "interactive", "i", false, "Create the workflow in interactive mode")

	cmd.RegisterFlagCompletionFunc("on-ddl", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			cobra.CompletionWithDesc("EXEC", "Try to execute the DDL on the target keyspace. If it fails, stop the workflow."),
			cobra.CompletionWithDesc("EXEC_IGNORE", "Try to execute the DDL on the target keyspace, ignoring any errors."),
			cobra.CompletionWithDesc("STOP", "Stop the workflow when a DDL statement is encountered."),
			cobra.CompletionWithDesc("IGNORE", "Ignore the DDL statement and continue the workflow."),
		}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func createInteractive(ctx context.Context, client *ps.Client, org, db, branch string, flags createFlags) error {
	return nil
}
