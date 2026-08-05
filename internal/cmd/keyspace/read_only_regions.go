package keyspace

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func ReadOnlyRegionsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-only-regions <database> <branch> <keyspace>",
		Short: "List read-only regions for a keyspace",
		Long: "List read-only regions configured for a Vitess keyspace.\n\n" +
			"This command is only supported for Vitess databases.",
		Args: cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch, keyspace := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching read-only regions for keyspace %s in %s/%s", printer.BoldBlue(keyspace), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			k, err := client.Keyspaces.Get(cmd.Context(), &ps.GetKeyspaceRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     keyspace,
				Full:         true,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("keyspace %s does not exist in branch %s (database: %s, organization: %s)", printer.BoldBlue(keyspace), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(k.ReadOnlyRegions) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Println("No read-only regions have been configured for this keyspace.")
				return nil
			}

			return ch.Printer.PrintResource(toReadOnlyRegions(k.ReadOnlyRegions))
		},
	}

	return cmd
}

type ReadOnlyRegion struct {
	Region      string `header:"region" json:"region"`
	ClusterSize string `header:"cluster_size" json:"cluster_name"`
	Replicas    int    `header:"replicas" json:"replicas"`

	orig *ps.ReadOnlyRegionKeyspace
}

func toReadOnlyRegions(regions []*ps.ReadOnlyRegionKeyspace) []*ReadOnlyRegion {
	out := make([]*ReadOnlyRegion, 0, len(regions))
	for _, region := range regions {
		out = append(out, &ReadOnlyRegion{
			Region:      region.Region,
			ClusterSize: region.ClusterDisplayName,
			Replicas:    region.Replicas,
			orig:        region,
		})
	}
	return out
}

func (r *ReadOnlyRegion) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", "  ")
}

func (r *ReadOnlyRegion) MarshalCSVValue() interface{} {
	return []*ReadOnlyRegion{r}
}
