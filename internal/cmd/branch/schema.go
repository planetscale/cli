package branch

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// SchemaCmd is the command for showing the schema of a branch.
func SchemaCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		web bool
	}

	cmd := &cobra.Command{
		Use:   "schema <database> <branch>",
		Short: "Show the schema of a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database, branch := args[0], args[1]

			if flags.web {
				ch.Printer.Println("🌐  Redirecting you to your branch schema in your web browser.")
				return browser.OpenURL(fmt.Sprintf("%s/%s/%s/%s/schema", cmdutil.ApplicationURL, ch.Config.Organization, database, branch))
			}
			client, err := ch.Client()
			if err != nil {
				return err
			}

			schemas, err := client.DatabaseBranches.Schema(ctx, &planetscale.BranchSchemaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)\n",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			if ch.Printer.Format() != printer.Human {
				return ch.Printer.PrintResource(schemas)
			}

			// human readable output
			for _, df := range schemas {
				ch.Printer.Println("--", printer.BoldBlue(df.Name), "--")
				scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(df.Raw)))
				for scanner.Scan() {
					txt := scanner.Text()
					if strings.HasPrefix(txt, "+") {
						ch.Printer.Println(color.New(color.FgGreen).Add(color.Bold).Sprint(txt)) //nolint: errcheck
					} else if strings.HasPrefix(txt, "-") {
						ch.Printer.Println(color.New(color.FgRed).Add(color.Bold).Sprint(txt)) //nolint: errcheck
					} else {
						ch.Printer.Println(txt)
					}
				}
				if err := scanner.Err(); err != nil {
					return fmt.Errorf("reading schema raw: %s", err)
				}
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&flags.web, "web", false, "Open in your web browser")

	return cmd
}
