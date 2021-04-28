package branch

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// DiffCmd is the command for showing the diff of a branch.
func DiffCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		web bool
	}

	cmd := &cobra.Command{
		Use:   "diff <database> <branch>",
		Short: "Show the diff of a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			diffs, err := client.DatabaseBranches.Diff(ctx, &planetscale.DiffBranchRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)\n",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if ch.Printer.Format() != printer.Human {
				return ch.Printer.PrintResource(diffs)
			}

			// human readable output
			for _, df := range diffs {
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
					return fmt.Errorf("reading diff raw: %s", err)
				}
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&flags.web, "web", false, "Open in your web browser")

	return cmd
}
