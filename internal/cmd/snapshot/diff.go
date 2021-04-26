package snapshot

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

// DiffCmd makes a command for fetching a single snapshot by its ID.
func DiffCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <snapshot-id>",
		Short: "Show the diff of a specific schema snapshot",
		Args:  cmdutil.RequiredArgs("snapshot-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			id := args[0]

			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching diff for schema snapshot %s", printer.BoldBlue(id)))
			defer end()

			diffs, err := client.SchemaSnapshots.Diff(ctx, &planetscale.DiffSchemaSnapshotRequest{
				ID: id,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("not diff found for snapshot id %s (organization: %s)", printer.BoldBlue(id), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}
			end()

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

	return cmd
}
