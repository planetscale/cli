package database

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// ReadOnlyRegionsCmd lists Vitess read-only regions for a database's default branch.
func ReadOnlyRegionsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-only-regions <database>",
		Short: "List Vitess read-only regions for a database",
		Long: "List Vitess read-only regions configured on the database's default branch.\n\n" +
			"This command is only supported for Vitess databases.",
		Args:    cmdutil.RequiredArgs("database"),
		Aliases: []string{"ror"},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return cmdutil.DatabaseCompletionFunc(ch, cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching read-only regions for %s...", printer.BoldBlue(database)))
			defer end()

			regions, err := client.ReadOnlyRegions.List(ctx, &ps.ListReadOnlyRegionsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(regions) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Println("No read-only regions have been configured for this database.")
				return nil
			}

			return ch.Printer.PrintResource(toReadOnlyRegions(regions))
		},
	}

	return cmd
}

// ReadOnlyRegions is a printable slice of read-only regions.
type ReadOnlyRegions []*ReadOnlyRegion

// ReadOnlyRegion is a table-serializable read-only region.
type ReadOnlyRegion struct {
	ID          string `header:"id" json:"id"`
	DisplayName string `header:"name" json:"display_name"`
	Region      string `header:"region" json:"region"`
	Ready       bool   `header:"ready" json:"ready"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`

	orig *ps.ReadOnlyRegion
}

func toReadOnlyRegion(r *ps.ReadOnlyRegion) *ReadOnlyRegion {
	return &ReadOnlyRegion{
		ID:          r.ID,
		DisplayName: r.DisplayName,
		Region:      r.Region.Slug,
		Ready:       r.Ready,
		CreatedAt:   r.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		orig:        r,
	}
}

func toReadOnlyRegions(regions []*ps.ReadOnlyRegion) ReadOnlyRegions {
	out := make([]*ReadOnlyRegion, 0, len(regions))
	for _, r := range regions {
		out = append(out, toReadOnlyRegion(r))
	}
	return out
}

func (r *ReadOnlyRegion) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *ReadOnlyRegion) MarshalCSVValue() interface{} {
	return []*ReadOnlyRegion{r}
}
