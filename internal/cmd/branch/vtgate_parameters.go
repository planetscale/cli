package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmd/vitessparams"
	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

var vtgateParameterNamespaces = []string{"vtgate"}

// VtgateParametersCmd lists the VTGate parameters of a Vitess branch.
func VtgateParametersCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:     "parameters <database> <branch>",
		Short:   "List the VTGate parameters of a Vitess branch",
		Long:    "List the VTGate parameters of a Vitess branch, including their current and default values.\n\nTo change parameters, use 'pscale branch vtgate update <database> <branch> --parameters vtgate.name=value'.",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"params"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching VTGate parameters for branch %s in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			parameters, err := client.DatabaseBranches.ListVTGateParameters(cmd.Context(), &ps.ListVTGateParametersRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return vtgateBranchError(ch, err, database, branch)
			}
			end()

			return ch.Printer.PrintResource(vitessparams.ToParameters(parameters))
		},
	}
}

// VtgateUpdateCmd changes the VTGate parameters of a Vitess branch.
func VtgateUpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		parameters []string
		resets     []string
	}

	cmd := &cobra.Command{
		Use:   "update <database> <branch>",
		Short: "Change the VTGate parameters of a Vitess branch",
		Long: `Change VTGate parameters on a Vitess branch. Pass each parameter as vtgate.name=value, and pass --reset vtgate.name to set a parameter back to its default.

The change is rolled out to the branch's VTGates. Use 'pscale branch vtgate changes list' to follow the rollout. To change the size or number of VTGates, use 'pscale branch vtgate resize'.`,
		Example: `  pscale branch vtgate update <database> <branch> \
    --parameters vtgate.max_memory_rows=500000 \
    --parameters vtgate.query-timeout=30000

  pscale branch vtgate update <database> <branch> --reset vtgate.query-timeout`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			changes, err := vitessparams.ParseChanges(flags.parameters, flags.resets, vtgateParameterNamespaces, "vtgate.max_memory_rows=500000")
			if err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Submitting VTGate parameter changes for branch %s in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			draft, err := client.DatabaseBranches.CreateVTGateConfigChange(ctx, &ps.CreateVTGateConfigChangeRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Options:      changes["vtgate"],
			})
			if err != nil {
				return vtgateBranchError(ch, err, database, branch)
			}

			if err := client.DatabaseBranches.SubmitConfigChanges(ctx, &ps.SubmitConfigChangesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				IDs:          []string{draft.ID},
			}); err != nil {
				_ = client.DatabaseBranches.CancelVTGateConfigChange(ctx, &ps.CancelVTGateConfigChangeRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Branch:       branch,
					ID:           draft.ID,
				})
				return vtgateBranchError(ch, err, database, branch)
			}

			change, err := client.DatabaseBranches.GetVTGateConfigChange(ctx, &ps.GetVTGateConfigChangeRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				ID:           draft.ID,
			})
			if err != nil {
				change = draft
			}
			end()

			return ch.Printer.PrintResource(vitessparams.ToConfigChange(change))
		},
	}

	cmd.Flags().StringArrayVar(&flags.parameters, "parameters", nil, "Set a parameter as vtgate.name=value (e.g. vtgate.max_memory_rows=500000). Repeatable. Use 'pscale branch vtgate parameters' to see available parameters.")
	cmd.Flags().StringArrayVar(&flags.resets, "reset", nil, "Set a parameter back to its default, as vtgate.name (e.g. vtgate.query-timeout). Repeatable.")

	return cmd
}

// VtgateChangesCmd manages VTGate parameter changes on a Vitess branch.
func VtgateChangesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changes <command>",
		Short: "Manage VTGate parameter changes for a Vitess branch",
	}
	cmd.AddCommand(vtgateChangesListCmd(ch), vtgateChangesShowCmd(ch), vtgateChangesCancelCmd(ch))
	return cmd
}

func vtgateChangesListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List VTGate parameter changes for a Vitess branch",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching VTGate parameter changes for branch %s in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			changes, err := client.DatabaseBranches.ListVTGateConfigChanges(cmd.Context(), &ps.ListVTGateConfigChangesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Page:         flags.page,
				PerPage:      flags.perPage,
			})
			if err != nil {
				return vtgateBranchError(ch, err, database, branch)
			}
			end()

			if len(changes) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No VTGate parameter changes found for branch %s in %s.\n", printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(vitessparams.ToConfigChanges(changes))
		},
	}

	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 25, "Number of results per page")
	return cmd
}

func vtgateChangesShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:     "show <database> <branch> <change-id>",
		Short:   "Show a VTGate parameter change for a Vitess branch",
		Args:    cmdutil.RequiredArgs("database", "branch", "change-id"),
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, changeID := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching VTGate parameter change %s for branch %s...", printer.BoldBlue(changeID), printer.BoldBlue(branch)))
			defer end()

			change, err := client.DatabaseBranches.GetVTGateConfigChange(cmd.Context(), &ps.GetVTGateConfigChangeRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				ID:           changeID,
			})
			if err != nil {
				if cmdutil.ErrCode(err) == ps.ErrNotFound {
					return vtgateChangeNotFoundError(changeID, database, branch)
				}
				return cmdutil.HandleError(err)
			}
			end()

			return ch.Printer.PrintResource(vitessparams.ToConfigChange(change))
		},
	}
}

func vtgateChangesCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <database> <branch> <change-id>",
		Short: "Cancel a pending VTGate parameter change for a Vitess branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "change-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, changeID := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Canceling VTGate parameter change %s for branch %s...", printer.BoldBlue(changeID), printer.BoldBlue(branch)))
			defer end()

			if err := client.DatabaseBranches.CancelVTGateConfigChange(cmd.Context(), &ps.CancelVTGateConfigChangeRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				ID:           changeID,
			}); err != nil {
				if cmdutil.ErrCode(err) == ps.ErrNotFound {
					return vtgateChangeNotFoundError(changeID, database, branch)
				}
				return cmdutil.HandleError(err)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Canceled VTGate parameter change %s for branch %s in %s.\n", printer.BoldBlue(changeID), printer.BoldBlue(branch), printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result":    "change canceled",
				"change_id": changeID,
			})
		},
	}
}

func vtgateBranchError(ch *cmdutil.Helper, err error, database, branch string) error {
	if cmdutil.ErrCode(err) == ps.ErrNotFound {
		return fmt.Errorf("database %s or branch %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
	}
	return cmdutil.HandleError(err)
}

func vtgateChangeNotFoundError(changeID, database, branch string) error {
	return fmt.Errorf("VTGate parameter change %s does not exist for branch %s in %s", printer.BoldBlue(changeID), printer.BoldBlue(branch), printer.BoldBlue(database))
}
