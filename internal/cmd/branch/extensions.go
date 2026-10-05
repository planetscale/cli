package branch

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ExtensionsCmd lists and toggles preloadable extensions on a Postgres branch.
func ExtensionsCmd(ch *cmdutil.Helper) *cobra.Command {
	long := `List extensions available on a Postgres branch's cluster image.

This is the catalog of extensions the image can load, not the result of
CREATE EXTENSION. Enable and disable change preload libraries through an
asynchronous branch change request.`

	run := func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		database, branch := args[0], args[1]

		client, err := ch.Client()
		if err != nil {
			return err
		}

		end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching extensions for branch %s in %s...", printer.BoldBlue(branch), printer.BoldBlue(database)))
		defer end()

		extensions, err := client.PostgresBranches.ListExtensions(ctx, &ps.ListPostgresExtensionsRequest{
			Organization: ch.Config.Organization,
			Database:     database,
			Branch:       branch,
		})
		if err != nil {
			switch cmdutil.ErrCode(err) {
			case ps.ErrNotFound:
				return fmt.Errorf("database %s or branch %s does not exist in organization %s",
					printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
			default:
				return cmdutil.HandleError(err)
			}
		}
		end()

		if len(extensions) == 0 && ch.Printer.Format() == printer.Human {
			ch.Printer.Printf("No extensions are listed for %s/%s.\n",
				printer.BoldBlue(database), printer.BoldBlue(branch))
			return nil
		}

		return ch.Printer.PrintResource(toPostgresExtensions(extensions))
	}

	cmd := &cobra.Command{
		Use:   "extensions <database> <branch>",
		Short: "List extensions available on a Postgres branch",
		Long:  long,
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE:  run,
	}

	listCmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List extensions available on a Postgres branch",
		Long:    long,
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE:    run,
	}
	cmd.AddCommand(listCmd, extensionToggleCmd(ch, true), extensionToggleCmd(ch, false))

	return cmd
}

func extensionToggleCmd(ch *cmdutil.Helper, enable bool) *cobra.Command {
	verb := "enable"
	if !enable {
		verb = "disable"
	}
	return &cobra.Command{
		Use:   verb + " <database> <branch> <extension>",
		Short: strings.ToUpper(verb[:1]) + verb[1:] + " a preloadable extension on a Postgres branch",
		Args:  cmdutil.ExactArgs("database", "branch", "extension"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, name := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			extensions, err := client.PostgresBranches.ListExtensions(cmd.Context(), &ps.ListPostgresExtensionsRequest{
				Organization: ch.Config.Organization, Database: database, Branch: branch,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			var extension *ps.PostgresExtension
			for _, candidate := range extensions {
				if candidate.Name == name {
					extension = candidate
					break
				}
			}
			if extension == nil {
				return fmt.Errorf("extension %s does not exist on branch %s", name, branch)
			}
			if extension.Internal || len(extension.Requirements) > 0 || extension.UnavailableReason != "" ||
				(extension.Loader != "shared_preload_libraries" && extension.Loader != "session_preload_libraries") {
				return fmt.Errorf("extension %s cannot be enabled or disabled", name)
			}

			includeInternal := false
			parameters, err := client.PostgresBranches.ListParameters(cmd.Context(), &ps.ListPostgresParametersRequest{
				Organization: ch.Config.Organization, Database: database, Branch: branch, Internal: &includeInternal,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			var loader *ps.PostgresParameter
			for _, parameter := range parameters {
				if parameter.Namespace == "pgconf" && parameter.Name == extension.Loader {
					loader = parameter
					break
				}
			}
			if loader == nil {
				return fmt.Errorf("extension %s cannot be enabled or disabled", name)
			}
			current := loader.Value
			if current == nil {
				current = loader.DefaultValue
			}
			selection, err := extensionNames(current)
			if err != nil {
				return fmt.Errorf("cannot read %s: %w", extension.Loader, err)
			}
			updated := make([]string, 0, len(selection)+1)
			found := false
			for _, selected := range selection {
				if selected == name {
					found = true
					if !enable {
						continue
					}
				}
				updated = append(updated, selected)
			}
			if enable && !found {
				updated = append(updated, name)
			}
			if found == enable {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Extension %s is already %sd on branch %s.\n", printer.BoldBlue(name), verb, printer.BoldBlue(branch))
					return nil
				}
				return ch.Printer.PrintResource(map[string]string{"result": "no_change", "extension": name, "branch": branch})
			}

			change, err := client.PostgresBranches.Resize(cmd.Context(), &ps.ResizePostgresBranchRequest{
				Organization: ch.Config.Organization, Database: database, Branch: branch,
				Parameters: map[string]map[string]string{"pgconf": {extension.Loader: strings.Join(updated, ",")}},
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}
			if change == nil {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Branch %s already matches the requested configuration.\n", printer.BoldBlue(branch))
					return nil
				}
				return ch.Printer.PrintResource(map[string]string{"result": "no_change", "extension": name, "branch": branch})
			}
			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Change to %s extension %s on branch %s %s (state: %s).\n", verb, printer.BoldBlue(name), printer.BoldBlue(branch), changeVerb(change.State), printer.BoldBlue(change.State))
				return nil
			}
			return ch.Printer.PrintResource(toPostgresBranchResize(change))
		},
	}
}

func extensionNames(value any) ([]string, error) {
	var names []string
	switch value := value.(type) {
	case nil:
		return nil, nil
	case string:
		names = strings.Split(value, ",")
	case []string:
		names = value
	case []any:
		for _, item := range value {
			name, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected library value %T", item)
			}
			names = append(names, name)
		}
	default:
		return nil, fmt.Errorf("unexpected library list %T", value)
	}
	selection := make([]string, 0, len(names))
	for _, name := range names {
		if name = strings.TrimSpace(name); name != "" {
			selection = append(selection, name)
		}
	}
	return selection, nil
}

type postgresExtension struct {
	Name              string `header:"name" json:"name"`
	Loader            string `header:"loader" json:"loader"`
	Available         bool   `header:"available" json:"available"`
	UnavailableReason string `header:"unavailable,n/a" json:"unavailable_reason"`
	URL               string `header:"url,n/a" json:"url"`

	orig *ps.PostgresExtension
}

func toPostgresExtensions(extensions []*ps.PostgresExtension) []*postgresExtension {
	out := make([]*postgresExtension, 0, len(extensions))
	for _, ext := range extensions {
		out = append(out, &postgresExtension{
			Name:              ext.Name,
			Loader:            ext.Loader,
			Available:         ext.Available,
			UnavailableReason: ext.UnavailableReason,
			URL:               ext.URL,
			orig:              ext,
		})
	}
	return out
}

func (e *postgresExtension) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(e.orig, "", "  ")
}

func (e *postgresExtension) MarshalCSVValue() interface{} {
	return []*postgresExtension{e}
}
