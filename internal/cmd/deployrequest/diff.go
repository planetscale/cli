package deployrequest

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// DiffCmd is the command for showing the diff of a deploy request.
func DiffCmd(cfg *config.Config) *cobra.Command {
	var flags struct {
		web bool
	}

	cmd := &cobra.Command{
		Use:   "diff <database> <number>",
		Short: "Show the diff of a deploy request",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database := args[0]
			number := args[1]

			if flags.web {
				fmt.Println("🌐  Redirecting you to your deploy request diff in your web browser.")
				return browser.OpenURL(fmt.Sprintf("%s/%s/%s/deploy-requests/%s/diff", cmdutil.ApplicationURL, cfg.Organization, database, number))
			}

			client, err := cfg.NewClientFromConfig()
			if err != nil {
				return err
			}

			n, err := strconv.ParseUint(number, 10, 64)
			if err != nil {
				return fmt.Errorf("The argument <number> is invalid: %s", err)
			}

			diffs, err := client.DeployRequests.Diff(ctx, &planetscale.DiffRequest{
				Organization: cfg.Organization,
				Database:     database,
				Number:       n,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("deploy rquest '%s/%s' does not exist in organization %s\n",
						cmdutil.BoldBlue(database), cmdutil.BoldBlue(number), cmdutil.BoldBlue(cfg.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			for _, df := range diffs {
				fmt.Println("--", cmdutil.BoldBlue(df.Name), "--")
				scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(df.Raw)))
				for scanner.Scan() {
					txt := scanner.Text()
					if strings.HasPrefix(txt, "+") {
						color.New(color.FgGreen).Add(color.Bold).Println(txt) //nolint: errcheck
					} else if strings.HasPrefix(txt, "-") {
						color.New(color.FgRed).Add(color.Bold).Println(txt) //nolint: errcheck
					} else {
						fmt.Println(txt)
					}
				}
				if err := scanner.Err(); err != nil {
					fmt.Fprintln(os.Stderr, "reading diff Raw:", err)
				}
			}
			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&flags.web, "web", false, "Open in your web browser")

	return cmd
}
