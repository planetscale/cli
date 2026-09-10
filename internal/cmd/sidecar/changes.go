package sidecar

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ChangesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changes <command>",
		Short: "Manage changes to a Neki sidecar",
	}
	cmd.AddCommand(changesListCmd(ch), changesShowCmd(ch), changesCancelCmd(ch))
	return cmd
}

func changesListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		period      string
		completedAt string
		page        int
		perPage     int
	}

	cmd := &cobra.Command{
		Use:     "list <database> <branch> <sidecar>",
		Short:   "List changes for a Neki sidecar",
		Args:    cmdutil.ExactArgs("database", "branch", "sidecar"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, sidecar := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching changes for sidecar %s", printer.BoldBlue(sidecar)))
			defer end()
			changes, err := client.NekiSidecars.ListChanges(cmd.Context(), &ps.ListNekiSidecarChangesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Sidecar:      sidecar,
				Period:       flags.period,
				CompletedAt:  flags.completedAt,
				Page:         flags.page,
				PerPage:      flags.perPage,
			})
			if err != nil {
				return handleError(err, database, branch, sidecar)
			}
			end()

			if len(changes) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No changes found for sidecar %s.\n", printer.BoldBlue(sidecar))
				return nil
			}
			return ch.Printer.PrintResource(toChanges(changes))
		},
	}

	cmd.Flags().StringVar(&flags.period, "period", "", "Only show changes from this period")
	cmd.Flags().StringVar(&flags.completedAt, "completed-at", "", "Only show changes completed at this time")
	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 100, "Number of results per page")
	return cmd
}

func changesShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:     "show <database> <branch> <sidecar> <change-id>",
		Short:   "Show a change to a Neki sidecar",
		Args:    cmdutil.ExactArgs("database", "branch", "sidecar", "change-id"),
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, sidecar, changeID := args[0], args[1], args[2], args[3]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching change %s", printer.BoldBlue(changeID)))
			defer end()
			change, err := client.NekiSidecars.GetChange(cmd.Context(), &ps.GetNekiSidecarChangeRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Sidecar:      sidecar,
				Change:       changeID,
			})
			if err != nil {
				return handleError(err, database, branch, sidecar)
			}
			end()
			if ch.Printer.Format() == printer.Human {
				if err := ch.Printer.PrintResource(toChange(change)); err != nil {
					return err
				}

				differences := changeDifferences(change)
				if len(differences) == 0 {
					ch.Printer.Println("No configuration differences.")
					return nil
				}

				ch.Printer.Println("Changes:")
				for _, difference := range differences {
					ch.Printer.Printf("  Parameter %s: %s → %s\n", difference.Field, difference.Before, difference.After)
				}
				return nil
			}
			return ch.Printer.PrintResource(toChange(change))
		},
	}
}

func changesCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <database> <branch> <sidecar> <change-id>",
		Short: "Cancel a pending change to a Neki sidecar",
		Args:  cmdutil.ExactArgs("database", "branch", "sidecar", "change-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, sidecar, changeID := args[0], args[1], args[2], args[3]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Cancelling change %s", printer.BoldBlue(changeID)))
			defer end()
			err = client.NekiSidecars.CancelChange(cmd.Context(), &ps.CancelNekiSidecarChangeRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Sidecar:      sidecar,
				Change:       changeID,
			})
			if err != nil {
				return handleError(err, database, branch, sidecar)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Canceled change %s for sidecar %s.\n", printer.BoldBlue(changeID), printer.BoldBlue(sidecar))
				return nil
			}
			return ch.Printer.PrintResource(map[string]string{
				"result":    "change canceled",
				"change_id": changeID,
				"sidecar":   sidecar,
			})
		},
	}
}
