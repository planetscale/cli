package keyspace

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func parametersChangesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changes <command>",
		Short: "Manage parameter changes to a keyspace",
	}
	cmd.AddCommand(parametersChangesListCmd(ch), parametersChangesShowCmd(ch), parametersChangesCancelCmd(ch))
	return cmd
}

func parametersChangesListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		page    int
		perPage int
	}

	cmd := &cobra.Command{
		Use:     "list <database> <branch> <keyspace>",
		Short:   "List parameter changes to a keyspace",
		Args:    cmdutil.RequiredArgs("database", "branch", "keyspace"),
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, keyspace := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching parameter changes for keyspace %s in %s/%s...", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			changes, err := client.Keyspaces.ListConfigChanges(cmd.Context(), &ps.ListKeyspaceConfigChangesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     keyspace,
				Page:         flags.page,
				PerPage:      flags.perPage,
			})
			if err != nil {
				return parameterChangeError(ch, err, database, branch, keyspace)
			}
			end()

			if len(changes) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No parameter changes found for keyspace %s in %s/%s.\n", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(toKeyspaceConfigChanges(changes))
		},
	}

	cmd.Flags().IntVar(&flags.page, "page", 0, "Page number to fetch")
	cmd.Flags().IntVar(&flags.perPage, "per-page", 25, "Number of results per page")
	return cmd
}

func parametersChangesShowCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:     "show <database> <branch> <keyspace> <change-id>",
		Short:   "Show a parameter change to a keyspace",
		Args:    cmdutil.RequiredArgs("database", "branch", "keyspace", "change-id"),
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, keyspace, changeID := args[0], args[1], args[2], args[3]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching parameter change %s for keyspace %s...", printer.BoldBlue(changeID), printer.BoldBlue(keyspace)))
			defer end()

			change, err := client.Keyspaces.GetConfigChange(cmd.Context(), &ps.GetKeyspaceConfigChangeRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     keyspace,
				ID:           changeID,
			})
			if err != nil {
				if cmdutil.ErrCode(err) == ps.ErrNotFound {
					return fmt.Errorf("parameter change %s does not exist for keyspace %s in %s/%s", printer.BoldBlue(changeID), printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch))
				}
				return cmdutil.HandleError(err)
			}
			end()

			return ch.Printer.PrintResource(toKeyspaceConfigChange(change))
		},
	}
}

func parametersChangesCancelCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <database> <branch> <keyspace> <change-id>",
		Short: "Cancel a pending parameter change to a keyspace",
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace", "change-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, keyspace, changeID := args[0], args[1], args[2], args[3]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Canceling parameter change %s for keyspace %s...", printer.BoldBlue(changeID), printer.BoldBlue(keyspace)))
			defer end()

			if err := client.Keyspaces.CancelConfigChange(cmd.Context(), &ps.CancelKeyspaceConfigChangeRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     keyspace,
				ID:           changeID,
			}); err != nil {
				if cmdutil.ErrCode(err) == ps.ErrNotFound {
					return fmt.Errorf("parameter change %s does not exist for keyspace %s in %s/%s", printer.BoldBlue(changeID), printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch))
				}
				return cmdutil.HandleError(err)
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Canceled parameter change %s for keyspace %s in %s/%s.\n", printer.BoldBlue(changeID), printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(map[string]string{
				"result":    "change canceled",
				"change_id": changeID,
			})
		},
	}
}

type keyspaceConfigChange struct {
	ID        string `header:"id" json:"id"`
	Component string `header:"component" json:"change_type"`
	State     string `header:"state" json:"state"`
	Changes   string `header:"changes" json:"changes"`
	CreatedAt string `header:"created at" json:"created_at"`

	orig *ps.KeyspaceConfigChange
}

func toKeyspaceConfigChange(change *ps.KeyspaceConfigChange) *keyspaceConfigChange {
	return &keyspaceConfigChange{
		ID:        change.ID,
		Component: change.ChangeType,
		State:     change.State,
		Changes:   formatParameterChanges(change),
		CreatedAt: change.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
		orig:      change,
	}
}

func toKeyspaceConfigChanges(changes []*ps.KeyspaceConfigChange) []*keyspaceConfigChange {
	out := make([]*keyspaceConfigChange, 0, len(changes))
	for _, change := range changes {
		out = append(out, toKeyspaceConfigChange(change))
	}
	return out
}

func (c *keyspaceConfigChange) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(c.orig, "", "  ")
}

func (c *keyspaceConfigChange) MarshalCSVValue() interface{} {
	return []*keyspaceConfigChange{c}
}

func formatParameterChanges(change *ps.KeyspaceConfigChange) string {
	names := make([]string, 0, len(change.NewOptions))
	for name := range change.NewOptions {
		names = append(names, name)
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		before := "(default)"
		if previous, ok := change.PreviousOptions[name]; ok && previous != nil {
			before = *previous
		}
		after := "(default)"
		if value := change.NewOptions[name]; value != nil {
			after = *value
		}
		parts = append(parts, fmt.Sprintf("%s: %s → %s", name, before, after))
	}
	return strings.Join(parts, ", ")
}
