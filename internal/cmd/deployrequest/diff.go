package deployrequest

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// DiffCmd is the command for showing the diff of a deploy request.
func DiffCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		web bool
	}

	cmd := &cobra.Command{
		Use:   "diff <database> <number>",
		Short: "Show the diff of a deploy request",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			number := args[1]

			if flags.web {
				ch.Printer.Println("🌐  Redirecting you to your deploy request diff in your web browser.")
				return browser.OpenURL(fmt.Sprintf("%s/%s/%s/deploy-requests/%s/diff", cmdutil.ApplicationURL, ch.Config.Organization, database, number))
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			n, err := strconv.ParseUint(number, 10, 64)
			if err != nil {
				return fmt.Errorf("the argument <number> is invalid: %s", err)
			}

			diffs, err := client.DeployRequests.Diff(ctx, &planetscale.DiffRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Number:       n,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("deploy request '%s/%s' does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(number), printer.BoldBlue(ch.Config.Organization))
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
