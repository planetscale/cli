package branch

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// ExtensionsCmd lists extensions available on a Postgres branch's cluster image.
func ExtensionsCmd(ch *cmdutil.Helper) *cobra.Command {
	long := `List extensions available on a Postgres branch's cluster image.

This is the catalog of extensions the image can load, not the result of
CREATE EXTENSION. There is no CLI command to enable an extension; preload
libraries are configured with 'pscale branch resize --parameters'.`

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
	cmd.AddCommand(listCmd)

	return cmd
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
