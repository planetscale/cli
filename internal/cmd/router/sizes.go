package router

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func SizesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sizes <database> <branch>",
		Short:   "List available router sizes for a Neki database branch",
		Args:    cmdutil.ExactArgs("database", "branch"),
		Aliases: []string{"size-skus"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database, branch := args[0], args[1]
			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching router sizes for %s/%s", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			skus, err := client.NekiRouters.ListSizeSKUs(cmd.Context(), &ps.ListNekiRouterSizeSKUsRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Rates:        true,
			})
			if err != nil {
				return handleError(err, database, branch, "")
			}
			end()

			if len(skus) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No router sizes found for %s/%s.\n", printer.BoldBlue(database), printer.BoldBlue(branch))
				return nil
			}

			return ch.Printer.PrintResource(toRouterSKUs(skus))
		},
	}

	return cmd
}

type routerSKUDisplay struct {
	Name   string `header:"name" json:"name"`
	CPU    string `header:"cpu" json:"cpu"`
	Memory string `header:"memory" json:"memory"`
	Cost   string `header:"cost" json:"rate"`

	orig *ps.NekiRouterSKU
}

func (s *routerSKUDisplay) MarshalJSON() ([]byte, error) { return json.Marshal(s.orig) }
func (s *routerSKUDisplay) MarshalCSVValue() interface{} { return []*routerSKUDisplay{s} }

func toRouterSKU(sku *ps.NekiRouterSKU) *routerSKUDisplay {
	name := sku.DisplayName
	if name == "" {
		name = cmdutil.ToClusterSizeSlug(sku.Name)
	}

	cost := ""
	if sku.Rate != nil && *sku.Rate > 0 {
		cost = fmt.Sprintf("$%d", *sku.Rate)
	}

	return &routerSKUDisplay{
		Name:   name,
		CPU:    sku.CPU,
		Memory: cmdutil.FormatParts(sku.RAM).IntString(),
		Cost:   cost,
		orig:   sku,
	}
}

func toRouterSKUs(skus []*ps.NekiRouterSKU) []*routerSKUDisplay {
	out := make([]*routerSKUDisplay, 0, len(skus))
	for _, sku := range skus {
		out = append(out, toRouterSKU(sku))
	}
	return out
}
