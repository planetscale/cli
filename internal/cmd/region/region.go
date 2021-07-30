package region

import (
	"encoding/json"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// RegionCmd encapsulates the commands for interacting with regions
func RegionCmd(ch *cmdutil.Helper) *cobra.Command {

	cmd := &cobra.Command{
		Use:               "region <command>",
		Short:             "List regions",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint: errcheck

	cmd.AddCommand(ListCmd(ch))

	return cmd
}

// Regions represents a slice of regions.
type Regions []*Region

type Region struct {
	Name    string `header:"name" json:"display_name"`
	Slug    string `header:"slug" json:"slug"`
	Enabled bool   `header:"enabled" json:"enabled"`

	orig *ps.Region
}

// toRegion returns a struct that prints out various fields of a region.
func toRegion(region *ps.Region) *Region {
	return &Region{
		Name:    region.Name,
		Slug:    region.Slug,
		Enabled: region.Enabled,
	}
}

func toRegions(regions []*ps.Region) Regions {
	rs := make([]*Region, 0, len(regions))

	for _, r := range regions {
		rs = append(rs, toRegion(r))
	}

	return rs
}

func (r *Region) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(r.orig, "", " ")
}

func (r *Region) MarshalCSVValue() interface{} {
	return []*Region{r}
}
