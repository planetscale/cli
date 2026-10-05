package branch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
request; only extensions marked "can enable" can be toggled. Toggling an
extension may restart the database.`

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
		// Exact args so a mistyped subcommand (e.g. "enabel db main vector")
		// errors instead of listing extensions for a database named "enabel".
		Args: cmdutil.ExactArgs("database", "branch"),
		RunE: run,
	}

	listCmd := &cobra.Command{
		Use:     "list <database> <branch>",
		Short:   "List extensions available on a Postgres branch",
		Long:    long,
		Args:    cmdutil.ExactArgs("database", "branch"),
		Aliases: []string{"ls"},
		RunE:    run,
	}
	cmd.AddCommand(listCmd, extensionToggleCmd(ch, true), extensionToggleCmd(ch, false))

	return cmd
}

func extensionToggleCmd(ch *cmdutil.Helper, enable bool) *cobra.Command {
	var flags struct {
		wait        bool
		waitTimeout time.Duration
	}

	verb := "enable"
	if !enable {
		verb = "disable"
	}
	cmd := &cobra.Command{
		Use:   verb + " <database> <branch> <extension>",
		Short: strings.ToUpper(verb[:1]) + verb[1:] + " an extension on a Postgres branch",
		Long: strings.ToUpper(verb[:1]) + verb[1:] + ` an extension on a Postgres branch.

This queues an asynchronous branch change request and may restart the
database. Only extensions marked "can enable" in 'pscale branch extensions
list' can be toggled.`,
		Args: cmdutil.ExactArgs("database", "branch", "extension"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, name := args[0], args[1], args[2]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s", printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			switch db.Kind {
			case ps.DatabaseEnginePostgres:
			case ps.DatabaseEngineNeki:
				return fmt.Errorf("extensions on Neki databases are set per configuration profile; use %s", printer.BoldBlue("pscale branch config-profile extensions "+verb))
			default:
				return fmt.Errorf("extensions are only available for PostgreSQL databases; %s is %s", printer.BoldBlue(database), printer.BoldBlue(string(db.Kind)))
			}

			extensions, err := client.PostgresBranches.ListExtensions(ctx, &ps.ListPostgresExtensionsRequest{
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
				} else if candidate.Enabled != nil && *candidate.Enabled {
					selection = append(selection, candidate.Name)
				}
			}
			if extension == nil {
				return fmt.Errorf("extension %s does not exist on branch %s", name, branch)
			}
			if !extension.CanEnable {
				return fmt.Errorf("extension %s cannot be enabled or disabled", name)
			}
			if extension.Enabled == nil {
				return fmt.Errorf("enabled state is unavailable for extension %s", name)
			}
			if *extension.Enabled == enable {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Extension %s is already %sd on branch %s.\n", printer.BoldBlue(name), verb, printer.BoldBlue(branch))
					return nil
				}
				return ch.Printer.PrintResource(map[string]string{"result": "no_change", "extension": name, "branch": branch})
			}
			if enable {
				selection = append(selection, name)
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Requesting to %s extension %s on branch %s...", verb, printer.BoldBlue(name), printer.BoldBlue(branch)))
			defer end()
			change, err := client.PostgresBranches.Resize(ctx, &ps.ResizePostgresBranchRequest{
				Organization: ch.Config.Organization, Database: database, Branch: branch,
				Extensions: &selection,
			})
			if err != nil {
				return extensionsError(ch, err, database, branch)
			}
			end()
			if change == nil {
				if ch.Printer.Format() == printer.Human {
					ch.Printer.Printf("Branch %s already matches the requested configuration.\n", printer.BoldBlue(branch))
					return nil
				}
				return ch.Printer.PrintResource(map[string]string{"result": "no_change", "extension": name, "branch": branch})
			}
			if flags.wait {
				change, err = waitForChange(ctx, ch, client, database, branch, change, flags.waitTimeout)
				if err != nil {
					return err
				}
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Change to %s extension %s on branch %s %s (state: %s).\n", verb, printer.BoldBlue(name), printer.BoldBlue(branch), changeVerb(change.State), printer.BoldBlue(change.State))
				// Once the change has finished, any restart already happened.
				if !change.Finished() {
					ch.Printer.Println("Note: this change may restart the database.")
				}
				return nil
			}
			return ch.Printer.PrintResource(toPostgresBranchResize(change))
		},
	}

	cmd.Flags().BoolVar(&flags.wait, "wait", false, "Wait for the change request to complete before returning.")
	cmd.Flags().DurationVar(&flags.waitTimeout, "wait-timeout", 10*time.Minute, "Maximum time to wait for the change request to complete with --wait.")

	return cmd
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
	Enabled   string `header:"enabled" json:"enabled"`
	CanEnable bool   `header:"can enable" json:"can_enable"`
	URL       string `header:"url,n/a" json:"url"`

	orig *ps.PostgresExtension
}

func toPostgresExtensions(extensions []*ps.PostgresExtension) []*postgresExtension {
	out := make([]*postgresExtension, 0, len(extensions))
	for _, ext := range extensions {
		enabled := "n/a"
		if ext.Enabled != nil {
			enabled = "No"
			if *ext.Enabled {
				enabled = "Yes"
			}
		}
		out = append(out, &postgresExtension{
			Name:      ext.Name,
			Enabled:   enabled,
			CanEnable: ext.CanEnable,
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
