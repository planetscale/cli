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

// ExtensionsCmd lists and toggles extensions on a Postgres branch.
func ExtensionsCmd(ch *cmdutil.Helper) *cobra.Command {
	long := `List extensions available on a Postgres branch's cluster image.

This is the catalog of extensions the image can load, not the result of
CREATE EXTENSION. Enable and disable queue an asynchronous branch change
request; only extensions marked "can enable" can be toggled.`

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
			return extensionsError(ch, err, database, branch)
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
		Short: strings.ToUpper(verb[:1]) + verb[1:] + " an extension on a Postgres branch",
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
				return extensionsError(ch, err, database, branch)
			}
			var extension *ps.PostgresExtension
			selection := []string{}
			for _, candidate := range extensions {
				if candidate.Name == name {
					extension = candidate
				} else if candidate.Enabled {
					selection = append(selection, candidate.Name)
				}
			}
			if extension == nil {
				return fmt.Errorf("extension %s does not exist on branch %s", name, branch)
			}
			if !extension.CanEnable {
				return fmt.Errorf("extension %s cannot be enabled or disabled", name)
			}
			if extension.Enabled == enable {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Extension %s is already %sd on branch %s.\n", printer.BoldBlue(name), verb, printer.BoldBlue(branch))
					return nil
				}
				return ch.Printer.PrintResource(map[string]string{"result": "no_change", "extension": name, "branch": branch})
			}
			if enable {
				selection = append(selection, name)
			}

			change, err := client.PostgresBranches.Resize(cmd.Context(), &ps.ResizePostgresBranchRequest{
				Organization: ch.Config.Organization, Database: database, Branch: branch,
				Extensions: &selection,
			})
			if err != nil {
				return extensionsError(ch, err, database, branch)
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

func extensionsError(ch *cmdutil.Helper, err error, database, branch string) error {
	if cmdutil.ErrCode(err) == ps.ErrNotFound {
		return fmt.Errorf("database %s or branch %s does not exist in organization %s",
			printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Organization))
	}
	return cmdutil.HandleError(err)
}

type postgresExtension struct {
	Name      string `header:"name" json:"name"`
	Enabled   bool   `header:"enabled" json:"enabled"`
	CanEnable bool   `header:"can enable" json:"can_enable"`
	Loader    string `header:"loader,n/a" json:"loader"`
	URL       string `header:"url,n/a" json:"url"`

	orig *ps.PostgresExtension
}

func toPostgresExtensions(extensions []*ps.PostgresExtension) []*postgresExtension {
	out := make([]*postgresExtension, 0, len(extensions))
	for _, ext := range extensions {
		out = append(out, &postgresExtension{
			Name:      ext.Name,
			Enabled:   ext.Enabled,
			CanEnable: ext.CanEnable,
			Loader:    ext.Loader,
			URL:       ext.URL,
			orig:      ext,
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
