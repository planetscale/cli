package keyspace

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func parametersSetCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		parameters []string
		resets     []string
	}

	cmd := &cobra.Command{
		Use:   "set <database> <branch> <keyspace>",
		Short: "Change the VTTablet and MySQL parameters of a keyspace",
		Long: `Change VTTablet or MySQL parameters on a keyspace. Pass each parameter as namespace.name=value, where namespace is vttablet or mysqld, and pass --reset namespace.name to set a parameter back to its default.

All changes are submitted together and rolled out to the keyspace. Use 'pscale keyspace parameters changes list' to follow the rollout.`,
		Example: `  pscale keyspace parameters set <database> <branch> <keyspace> \
    --parameters vttablet.vreplication-parallel-insert-workers=4 \
    --parameters vttablet.vreplication_max_time_to_retry_on_error=720h

  pscale keyspace parameters set <database> <branch> <keyspace> \
    --reset vttablet.vreplication-parallel-insert-workers`,
		Args: cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			changes, err := parseParameterChanges(flags.parameters, flags.resets)
			if err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Submitting parameter changes for keyspace %s in %s/%s...", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			drafts := make([]*ps.KeyspaceConfigChange, 0, len(changes))
			for _, namespace := range parameterNamespaces {
				options, ok := changes[namespace]
				if !ok {
					continue
				}

				draft, err := client.Keyspaces.CreateConfigChange(ctx, &ps.CreateKeyspaceConfigChangeRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Branch:       branch,
					Keyspace:     keyspace,
					ChangeType:   namespace,
					Options:      options,
				})
				if err != nil {
					cancelDrafts(ctx, client, ch.Config.Organization, database, branch, keyspace, drafts)
					return parameterChangeError(ch, err, database, branch, keyspace)
				}
				drafts = append(drafts, draft)
			}

			ids := make([]string, 0, len(drafts))
			for _, draft := range drafts {
				ids = append(ids, draft.ID)
			}

			if err := client.Keyspaces.SubmitConfigChanges(ctx, &ps.SubmitConfigChangesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				IDs:          ids,
			}); err != nil {
				cancelDrafts(ctx, client, ch.Config.Organization, database, branch, keyspace, drafts)
				return parameterChangeError(ch, err, database, branch, keyspace)
			}

			submitted := make([]*ps.KeyspaceConfigChange, 0, len(drafts))
			for _, draft := range drafts {
				change, err := client.Keyspaces.GetConfigChange(ctx, &ps.GetKeyspaceConfigChangeRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Branch:       branch,
					Keyspace:     keyspace,
					ID:           draft.ID,
				})
				if err != nil {
					change = draft
				}
				submitted = append(submitted, change)
			}
			end()

			return ch.Printer.PrintResource(toKeyspaceConfigChanges(submitted))
		},
	}

	cmd.Flags().StringArrayVar(&flags.parameters, "parameters", nil, "Set a parameter as namespace.name=value, where namespace is vttablet or mysqld (e.g. vttablet.vreplication-parallel-insert-workers=4). Repeatable. Use 'pscale keyspace parameters list' to see available parameters.")
	cmd.Flags().StringArrayVar(&flags.resets, "reset", nil, "Set a parameter back to its default, as namespace.name (e.g. vttablet.vreplication-parallel-insert-workers). Repeatable.")

	return cmd
}

// parseParameterChanges groups --parameters and --reset values by namespace.
// A nil value resets the parameter to its default.
func parseParameterChanges(sets, resets []string) (map[string]map[string]*string, error) {
	if len(sets) == 0 && len(resets) == 0 {
		return nil, fmt.Errorf("pass at least one --parameters namespace.name=value or --reset namespace.name")
	}

	changes := make(map[string]map[string]*string)
	add := func(flag, raw, key string, value *string) error {
		namespace, name, found := strings.Cut(key, ".")
		if !found || namespace == "" || name == "" {
			return fmt.Errorf("invalid %s %q: parameter must be prefixed with its namespace, e.g. vttablet.%s", flag, raw, key)
		}
		if !slices.Contains(parameterNamespaces, namespace) {
			return fmt.Errorf("invalid %s %q: namespace must be one of: %s", flag, raw, strings.Join(parameterNamespaces, ", "))
		}
		if _, exists := changes[namespace][name]; exists {
			return fmt.Errorf("parameter %s.%s is passed more than once", namespace, name)
		}
		if changes[namespace] == nil {
			changes[namespace] = make(map[string]*string)
		}
		changes[namespace][name] = value
		return nil
	}

	for _, set := range sets {
		key, value, found := strings.Cut(set, "=")
		if !found {
			return nil, fmt.Errorf("invalid --parameters %q: expected namespace.name=value (e.g. vttablet.vreplication-parallel-insert-workers=4)", set)
		}
		if err := add("--parameters", set, key, &value); err != nil {
			return nil, err
		}
	}

	for _, reset := range resets {
		if err := add("--reset", reset, reset, nil); err != nil {
			return nil, err
		}
	}

	return changes, nil
}

func cancelDrafts(ctx context.Context, client *ps.Client, organization, database, branch, keyspace string, drafts []*ps.KeyspaceConfigChange) {
	for _, draft := range drafts {
		_ = client.Keyspaces.CancelConfigChange(ctx, &ps.CancelKeyspaceConfigChangeRequest{
			Organization: organization,
			Database:     database,
			Branch:       branch,
			Keyspace:     keyspace,
			ID:           draft.ID,
		})
	}
}

func parameterChangeError(ch *cmdutil.Helper, err error, database, branch, keyspace string) error {
	if cmdutil.ErrCode(err) == ps.ErrNotFound {
		return keyspaceNotFoundError(ch, database, branch, keyspace)
	}
	return cmdutil.HandleError(err)
}
